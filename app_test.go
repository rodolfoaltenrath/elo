package main

import (
	"slices"
	"strings"
	"testing"
)

func TestLinuxSupportDetectsFedora(t *testing.T) {
	support := linuxSupportFromOSRelease("linux", map[string]string{
		"ID":          "fedora",
		"PRETTY_NAME": "Fedora Linux 43 (Workstation Edition)",
	})

	if !support.Supported {
		t.Fatal("Fedora deveria ter automação habilitada")
	}
	if support.Family != "Fedora" {
		t.Fatalf("Family = %q, want Fedora", support.Family)
	}
	if support.PackageManager != "dnf" {
		t.Fatalf("PackageManager = %q, want dnf", support.PackageManager)
	}
}

func TestLinuxSupportDetectsFedoraDerivative(t *testing.T) {
	support := linuxSupportFromOSRelease("linux", map[string]string{
		"ID":          "nobara",
		"ID_LIKE":     "fedora",
		"PRETTY_NAME": "Nobara Linux",
	})

	if !support.Supported || support.PackageManager != "dnf" {
		t.Fatalf("derivado Fedora detectado incorretamente: %+v", support)
	}
}

func TestLinuxSupportPreservesExistingFamilies(t *testing.T) {
	tests := []struct {
		name    string
		values  map[string]string
		manager string
	}{
		{
			name:    "Arch",
			values:  map[string]string{"ID": "cachyos", "ID_LIKE": "arch"},
			manager: "pacman",
		},
		{
			name:    "Debian",
			values:  map[string]string{"ID": "ubuntu", "ID_LIKE": "debian"},
			manager: "apt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			support := linuxSupportFromOSRelease("linux", tt.values)
			if !support.Supported || support.PackageManager != tt.manager {
				t.Fatalf("distribuição detectada incorretamente: %+v", support)
			}
		})
	}
}

func TestFedoraPackageMapping(t *testing.T) {
	support := linuxSupport{PackageManager: "dnf", Supported: true}
	got := requiredPackageNames(support)
	want := []string{"pcsc-lite-ccid", "opensc", "pcsc-lite", "pcsc-tools", "nss-tools"}

	if !slices.Equal(got, want) {
		t.Fatalf("requiredPackageNames() = %v, want %v", got, want)
	}
}

func TestDnfInstallCommandQuotesPackageNames(t *testing.T) {
	support := linuxSupport{PackageManager: "dnf", Supported: true}
	commands := packageInstallCommands(support, []string{"opensc", "nss-tools"})

	want := "dnf install -y 'opensc' 'nss-tools'"
	if len(commands) != 1 || commands[0] != want {
		t.Fatalf("packageInstallCommands() = %v, want %q", commands, want)
	}
}

func TestFedoraAutoFixUsesDNFAndJava8(t *testing.T) {
	support := linuxSupport{PackageManager: "dnf", Supported: true}
	script := autoFixScript(support)

	for _, expected := range []string{
		"dnf install -y",
		"'pcsc-lite-ccid'",
		"'nss-tools'",
		"'icedtea-web'",
		"java-1.8.0-openjdk",
		"temurin-8-jre",
		"gpgcheck=1",
		"systemctl enable --now pcscd.socket",
	} {
		if !strings.Contains(script, expected) {
			t.Fatalf("script Fedora não contém %q:\n%s", expected, script)
		}
	}
	if strings.Contains(script, "pacman ") || strings.Contains(script, "apt-get ") {
		t.Fatalf("script Fedora contém outro gerenciador de pacotes:\n%s", script)
	}
}

func TestFedoraRPMDriverUsesDNF(t *testing.T) {
	support := linuxSupport{PackageManager: "dnf", Supported: true}
	script := driverInstallScript(support, "/tmp/driver token.rpm", "rpm")

	if !strings.Contains(script, "dnf install -y '/tmp/driver token.rpm'") {
		t.Fatalf("instalador RPM não usa dnf:\n%s", script)
	}
	if strings.Contains(script, "bsdtar -xf '/tmp/driver token.rpm'") {
		t.Fatalf("instalador RPM do Fedora não deve extrair o pacote diretamente:\n%s", script)
	}
}

func TestFedoraPKCS11PathsIncludeLib64(t *testing.T) {
	if !slices.Contains(pkcs11ModuleCandidates, "/usr/lib64/pkcs11/opensc-pkcs11.so") {
		t.Fatal("caminho PKCS#11 padrão do Fedora não foi incluído")
	}
}
