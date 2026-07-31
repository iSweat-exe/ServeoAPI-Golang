package firewall

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"serveoapi/internal/core/config"
)

// Service pilote le firewall UFW de l'hôte via le binaire `ufw`. Le processus
// doit tourner avec les privilèges nécessaires (root / CAP_NET_ADMIN) et `ufw`
// doit être installé sur le système — sinon toutes les méthodes renvoient
// ErrUFWNotInstalled.
type Service struct {
	Config *config.Config
}

var (
	// ErrUFWNotInstalled indique que le binaire `ufw` est absent du PATH.
	ErrUFWNotInstalled = errors.New("ufw is not installed on this server")
	// ErrProtectedRule indique qu'une règle protège l'accès SSH ou l'API elle-même.
	ErrProtectedRule = errors.New(
		"this rule protects SSH or API access and cannot be modified",
	)
	// ErrWouldLockout indique qu'activer le firewall maintenant couperait
	// l'accès SSH ou l'accès à l'API (aucune règle allow correspondante).
	ErrWouldLockout = errors.New(
		"enabling the firewall now would block SSH or API access: add an allow rule first",
	)
)

func (s *Service) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "ufw", args...)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", ErrUFWNotInstalled
		}
		if output != "" {
			return "", fmt.Errorf("%s: %w", output, err)
		}
		return "", err
	}
	return output, nil
}

// Status retourne l'état actuel d'UFW (actif ou non) et ses règles numérotées.
func (s *Service) Status(ctx context.Context) (StatusResponse, error) {
	out, err := s.run(ctx, "status", "numbered")
	if err != nil {
		return StatusResponse{}, err
	}
	return parseStatus(out), nil
}

// Enable active UFW, sauf si cela couperait immédiatement l'accès SSH ou API
// (politique par défaut "deny" en entrée sans règle allow correspondante).
func (s *Service) Enable(ctx context.Context) (string, error) {
	status, err := s.Status(ctx)
	if err != nil {
		return "", err
	}

	if !status.Active {
		policy, err := s.defaultIncomingPolicy(ctx)
		if err != nil {
			return "", err
		}
		if policy == "deny" || policy == "reject" {
			apiPort, _ := strconv.Atoi(s.Config.Port)
			hasSSH := hasAllowRuleFor(status.Rules, 22)
			hasAPI := apiPort <= 0 || hasAllowRuleFor(status.Rules, apiPort)
			if !hasSSH || !hasAPI {
				return "", ErrWouldLockout
			}
		}
	}

	return s.run(ctx, "--force", "enable")
}

// Disable désactive UFW (laisse passer tout le trafic).
func (s *Service) Disable(ctx context.Context) (string, error) {
	return s.run(ctx, "--force", "disable")
}

// AddRule valide puis crée une nouvelle règle UFW. Refuse les règles deny/reject
// qui bloqueraient explicitement le port SSH (22) ou le port de l'API.
func (s *Service) AddRule(ctx context.Context, req AddRuleRequest) (string, error) {
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if !actionWhitelist[action] {
		return "", fmt.Errorf("action must be one of allow, deny, reject, limit")
	}

	port, err := validatePort(req.Port)
	if err != nil {
		return "", err
	}

	protocol := strings.ToLower(strings.TrimSpace(req.Protocol))
	if protocol != "" && protocol != "tcp" && protocol != "udp" {
		return "", fmt.Errorf("protocol must be 'tcp', 'udp' or empty")
	}

	from, err := validateFrom(req.From)
	if err != nil {
		return "", err
	}

	apiPort, _ := strconv.Atoi(s.Config.Port)
	blocksEveryone := (action == "deny" || action == "reject") && from == "any"
	if blocksEveryone &&
		(portRangeContains(port, 22) || (apiPort > 0 && portRangeContains(port, apiPort))) {
		return "", ErrProtectedRule
	}

	args := []string{"--force", action}
	if protocol != "" {
		args = append(args, "proto", protocol)
	}
	args = append(args, "from", from, "to", "any", "port", port)
	if comment := sanitizeComment(req.Comment); comment != "" {
		args = append(args, "comment", comment)
	}

	return s.run(ctx, args...)
}

// DeleteRule supprime une règle par son numéro (voir Status). Refuse de
// supprimer la règle qui protège l'accès SSH ou le port de l'API.
func (s *Service) DeleteRule(ctx context.Context, number int) (string, error) {
	if number <= 0 {
		return "", fmt.Errorf("invalid rule number")
	}

	status, err := s.Status(ctx)
	if err != nil {
		return "", err
	}

	apiPort, _ := strconv.Atoi(s.Config.Port)
	for _, rule := range status.Rules {
		if rule.Number != number {
			continue
		}
		if ruleCoversPort(rule.To, 22) || (apiPort > 0 && ruleCoversPort(rule.To, apiPort)) {
			return "", ErrProtectedRule
		}
		break
	}

	return s.run(ctx, "--force", "delete", strconv.Itoa(number))
}

func (s *Service) defaultIncomingPolicy(ctx context.Context) (string, error) {
	out, err := s.run(ctx, "status", "verbose")
	if err != nil {
		return "", err
	}
	match := defaultPolicyRe.FindStringSubmatch(out)
	if match == nil {
		return "", nil
	}
	return strings.ToLower(match[1]), nil
}

// Parsing helpers

var (
	ruleLineRe      = regexp.MustCompile(`^\[\s*(\d+)\]\s*(.+)$`)
	columnSplitRe   = regexp.MustCompile(`\s{2,}`)
	defaultPolicyRe = regexp.MustCompile(`(?i)Default:\s*(\w+)\s*\(incoming\)`)
	portSpecRe      = regexp.MustCompile(`^\d{1,5}(:\d{1,5})?$`)

	actionWhitelist = map[string]bool{"allow": true, "deny": true, "reject": true, "limit": true}
)

func parseStatus(output string) StatusResponse {
	status := StatusResponse{Rules: []Rule{}}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		trimmed := strings.TrimSpace(scanner.Text())
		switch trimmed {
		case "Status: active":
			status.Active = true
			continue
		case "Status: inactive":
			status.Active = false
			continue
		}

		match := ruleLineRe.FindStringSubmatch(trimmed)
		if match == nil {
			continue
		}
		number, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}

		cols := columnSplitRe.Split(strings.TrimSpace(match[2]), -1)
		rule := Rule{Number: number, From: "Anywhere"}
		if len(cols) > 0 {
			rule.To = cols[0]
		}
		if len(cols) > 1 {
			rule.Action = cols[1]
		}
		if len(cols) > 2 {
			rule.From = cols[2]
		}
		status.Rules = append(status.Rules, rule)
	}
	return status
}

func hasAllowRuleFor(rules []Rule, port int) bool {
	for _, r := range rules {
		if ruleIsAllow(r.Action) && ruleCoversPort(r.To, port) {
			return true
		}
	}
	return false
}

func ruleIsAllow(action string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(action)), "ALLOW")
}

// ruleCoversPort indique si la colonne "To" d'une règle UFW (ex: "22/tcp",
// "8080", "OpenSSH", "22/tcp (v6)") couvre le port donné.
func ruleCoversPort(to string, port int) bool {
	normalized := strings.ToLower(strings.TrimSpace(to))
	normalized = strings.TrimSuffix(normalized, " (v6)")
	if port == 22 && (strings.Contains(normalized, "openssh") || normalized == "ssh") {
		return true
	}
	portPart, _, _ := strings.Cut(normalized, "/")
	return portRangeContains(portPart, port)
}

// portRangeContains indique si la spec "N" ou "N:M" contient le port cible.
func portRangeContains(spec string, target int) bool {
	spec = strings.TrimSpace(spec)
	parts := strings.SplitN(spec, ":", 2)
	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	end := start
	if len(parts) == 2 {
		end, err = strconv.Atoi(parts[1])
		if err != nil {
			return false
		}
	}
	return target >= start && target <= end
}

func validatePort(port string) (string, error) {
	port = strings.TrimSpace(port)
	if !portSpecRe.MatchString(port) {
		return "", fmt.Errorf("port must be a number (1-65535) or a range like 8000:9000")
	}
	parts := strings.SplitN(port, ":", 2)
	values := make([]int, 0, len(parts))
	for _, p := range parts {
		n, _ := strconv.Atoi(p)
		if n < 1 || n > 65535 {
			return "", fmt.Errorf("port %d is out of range (1-65535)", n)
		}
		values = append(values, n)
	}
	if len(values) == 2 && values[0] > values[1] {
		return "", fmt.Errorf("invalid port range: %s", port)
	}
	return port, nil
}

func validateFrom(from string) (string, error) {
	from = strings.TrimSpace(from)
	if from == "" || strings.EqualFold(from, "any") {
		return "any", nil
	}
	if ip := net.ParseIP(from); ip != nil {
		return from, nil
	}
	if _, _, err := net.ParseCIDR(from); err == nil {
		return from, nil
	}
	return "", fmt.Errorf("from must be 'any', an IP address or a CIDR block")
}

func sanitizeComment(comment string) string {
	comment = strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			return ' '
		case r < 32:
			return -1
		default:
			return r
		}
	}, comment)
	comment = strings.TrimSpace(comment)
	const maxLen = 100
	if len(comment) > maxLen {
		comment = comment[:maxLen]
	}
	return comment
}
