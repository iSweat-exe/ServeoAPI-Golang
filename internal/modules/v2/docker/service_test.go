package docker

import "testing"

func TestIsBindSourceAllowed(t *testing.T) {
	const allowedRoot = "/var/serveoapi/data/"

	cases := []struct {
		name string
		src  string
		want bool
	}{
		{"root itself", "/var/serveoapi/data", true},
		{"direct child", "/var/serveoapi/data/minecraft", true},
		{"nested child", "/var/serveoapi/data/minecraft/world", true},
		{"redundant separators", "/var/serveoapi/data//minecraft/", true},
		{"inner traversal staying inside", "/var/serveoapi/data/a/../b", true},

		{"traversal escaping the root", "/var/serveoapi/data/../../../etc", false},
		{"traversal to docker socket", "/var/serveoapi/data/../../run/docker.sock", false},
		{"sibling directory with shared prefix", "/var/serveoapi/database", false},
		{"unrelated absolute path", "/etc/shadow", false},
		{"relative path", "data/minecraft", false},
		{"empty path", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isBindSourceAllowed(tc.src, allowedRoot); got != tc.want {
				t.Fatalf(
					"isBindSourceAllowed(%q, %q) = %v, want %v",
					tc.src,
					allowedRoot,
					got,
					tc.want,
				)
			}
		})
	}
}

// Sur Windows, les chemins hôtes fournis par Docker utilisent des antislashs :
// la normalisation doit les traiter comme des séparateurs.
func TestIsBindSourceAllowedBackslashTraversal(t *testing.T) {
	if isBindSourceAllowed(`/var/serveoapi/data\..\..\etc`, "/var/serveoapi/data/") {
		t.Fatal("backslash traversal must be rejected")
	}
}

func TestValidateComposeVolumesRejectsTraversal(t *testing.T) {
	const allowedRoot = "/var/serveoapi/data/"

	compose := map[string]interface{}{
		"services": map[string]interface{}{
			"app": map[string]interface{}{
				"volumes": []interface{}{
					"/var/serveoapi/data/../../../etc/passwd:/etc/passwd",
				},
			},
		},
	}

	if err := validateComposeVolumes(compose, allowedRoot); err == nil {
		t.Fatal("expected bind-mount traversal to be rejected")
	}
}

func TestValidateComposeVolumesAllowsNamedAndSafeBinds(t *testing.T) {
	const allowedRoot = "/var/serveoapi/data/"

	compose := map[string]interface{}{
		"services": map[string]interface{}{
			"app": map[string]interface{}{
				"volumes": []interface{}{
					"pgdata:/var/lib/postgresql/data",
					"/var/serveoapi/data/minecraft:/data",
					map[string]interface{}{
						"source": "/var/serveoapi/data/config",
						"target": "/config",
					},
				},
			},
		},
	}

	if err := validateComposeVolumes(compose, allowedRoot); err != nil {
		t.Fatalf("unexpected rejection: %v", err)
	}
}
