package config

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

// CaddyfileGenerator generates Caddyfile configurations.
type CaddyfileGenerator struct {
	configMgr *Manager
	homeDir   string
}

// NewCaddyfileGenerator creates a new Caddyfile generator.
func NewCaddyfileGenerator(configMgr *Manager, homeDir string) *CaddyfileGenerator {
	return &CaddyfileGenerator{
		configMgr: configMgr,
		homeDir:   homeDir,
	}
}

// CaddyTemplateData holds data for the Caddyfile template.
type CaddyTemplateData struct {
	BirdnetpiURL string
	HasPassword  bool
	PasswordHash string
	AppDir       string
	ExtractedDir string
}

// caddyfileTemplate is the template for generating Caddyfiles.
const caddyfileTemplate = `http:// {{.BirdnetpiURL}} {
  encode gzip

  # Go API endpoints
  handle /api/* {
    reverse_proxy localhost:8080
  }

  # WebSocket endpoint
  handle /ws {
    reverse_proxy localhost:8080
  }

  # Bird recordings (browse)
  handle /By_Date/* {
    root * {{.ExtractedDir}}
    file_server browse
  }
  handle /Charts/* {
    root * {{.ExtractedDir}}
    file_server browse
  }

  # Live spectrogram image
  handle /spectrogram.png {
    root * {{.ExtractedDir}}
    file_server
  }

  # Stream{{if .HasPassword}} (protected){{end}}
  handle /stream {
    {{- if .HasPassword}}
    basicauth {
      birdnet {{.PasswordHash}}
    }
    {{- end}}
    reverse_proxy localhost:8000
  }

  # Terminal{{if .HasPassword}} (protected){{end}}
  handle /terminal* {
    {{- if .HasPassword}}
    basicauth {
      birdnet {{.PasswordHash}}
    }
    {{- end}}
    reverse_proxy localhost:8888
  }

  # Preact SPA (default handler)
  handle {
    root * {{.AppDir}}

    @static path *.js *.css *.woff *.woff2 *.ttf
    header @static Cache-Control "public, max-age=31536000, immutable"

    @html path *.html /
    header @html Cache-Control "no-cache, no-store, must-revalidate"

    try_files {path} /index.html
    file_server
  }
}
`

// Generate generates and writes a new Caddyfile based on current config.
func (g *CaddyfileGenerator) Generate() error {
	cfg := g.configMgr.Get()

	// Derive paths
	userHome := g.homeDir
	if cfg.RecsDir != "" {
		// Extract home from RECS_DIR (e.g., /home/addlatt/BirdSongs -> /home/addlatt)
		userHome = filepath.Dir(cfg.RecsDir)
	}

	appDir := filepath.Join(userHome, "BirdNET-Pi", "web", "dist")
	extractedDir := cfg.Extracted
	if extractedDir == "" {
		extractedDir = filepath.Join(cfg.RecsDir, "Extracted")
	}

	// Prepare template data
	data := CaddyTemplateData{
		BirdnetpiURL: cfg.BirdnetpiURL,
		HasPassword:  cfg.CaddyPwd != "",
		AppDir:       appDir,
		ExtractedDir: extractedDir,
	}

	// Generate password hash if password is set
	if cfg.CaddyPwd != "" {
		hash, err := g.hashPassword(cfg.CaddyPwd)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		data.PasswordHash = hash
	}

	// Parse and execute template
	tmpl, err := template.New("caddyfile").Parse(caddyfileTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Backup existing Caddyfile
	caddyfilePath := "/etc/caddy/Caddyfile"
	if _, err := os.Stat(caddyfilePath); err == nil {
		if err := copyFile(caddyfilePath, caddyfilePath+".original"); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: failed to backup Caddyfile: %v\n", err)
		}
	}

	// Ensure directory exists
	if err := os.MkdirAll("/etc/caddy", 0755); err != nil {
		return fmt.Errorf("failed to create caddy directory: %w", err)
	}

	// Write new Caddyfile
	if err := os.WriteFile(caddyfilePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write Caddyfile: %w", err)
	}

	// Format the Caddyfile using caddy fmt
	if err := exec.Command("sudo", "caddy", "fmt", "--overwrite", caddyfilePath).Run(); err != nil {
		// Log but don't fail - the file is still valid
		fmt.Printf("Warning: failed to format Caddyfile: %v\n", err)
	}

	// Reload Caddy
	if err := exec.Command("sudo", "systemctl", "reload", "caddy").Run(); err != nil {
		return fmt.Errorf("failed to reload Caddy: %w", err)
	}

	return nil
}

// hashPassword uses caddy hash-password to generate a bcrypt hash.
func (g *CaddyfileGenerator) hashPassword(password string) (string, error) {
	cmd := exec.Command("caddy", "hash-password", "--plaintext", password)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(output)), nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, content, 0644)
}
