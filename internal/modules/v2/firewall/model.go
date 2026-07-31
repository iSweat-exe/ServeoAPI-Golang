package firewall

// Rule représente une règle UFW telle que rapportée par `ufw status numbered`.
type Rule struct {
	Number int    `json:"number"`
	To     string `json:"to"`
	Action string `json:"action"`
	From   string `json:"from"`
}

// StatusResponse décrit l'état courant du firewall UFW de l'hôte.
type StatusResponse struct {
	Active bool   `json:"active"`
	Rules  []Rule `json:"rules"`
}

// AddRuleRequest est le payload accepté pour créer une nouvelle règle UFW.
type AddRuleRequest struct {
	// Action: "allow", "deny", "reject" ou "limit".
	Action string `json:"action"`
	// Port: un port ("8080") ou une plage ("8000:9000").
	Port string `json:"port"`
	// Protocol: "tcp", "udp", ou vide pour les deux.
	Protocol string `json:"protocol"`
	// From: IP ou CIDR autorisé/visé, "any" par défaut.
	From    string `json:"from"`
	Comment string `json:"comment"`
}
