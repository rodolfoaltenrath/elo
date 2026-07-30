package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	javaBase              = "/usr/lib/jvm/java-8-openjdk"
	eloPKCS11Dir          = "/usr/local/lib/elo-pkcs11"
	eloUdevRule           = "/etc/udev/rules.d/99-elo-smartcard.rules"
	starSignTokenUdevRule = `SUBSYSTEM=="usb", ENV{DEVTYPE}=="usb_device", ATTR{idVendor}=="1059", ATTR{idProduct}=="0019", GROUP="pcscd", MODE="0660", TAG+="uaccess"`
)

type packageSpec struct {
	ID       string
	Label    string
	Arch     string
	Debian   string
	Fedora   string
	Required bool
}

type linuxSupport struct {
	ID             string
	Name           string
	Family         string
	PackageManager string
	Supported      bool
}

var requiredPackages = []packageSpec{
	{ID: "ccid", Label: "Driver do leitor", Arch: "ccid", Debian: "libccid", Fedora: "pcsc-lite-ccid", Required: true},
	{ID: "opensc", Label: "Leitura do certificado", Arch: "opensc", Debian: "opensc", Fedora: "opensc", Required: true},
	{ID: "pcsc", Label: "Comunicação PC/SC", Arch: "pcsclite", Debian: "pcscd", Fedora: "pcsc-lite", Required: true},
	{ID: "pcsc-tools", Label: "Ferramentas do token", Arch: "pcsc-tools", Debian: "pcsc-tools", Fedora: "pcsc-tools", Required: false},
	{ID: "nss-tools", Label: "Integração com navegador", Arch: "nss", Debian: "libnss3-tools", Fedora: "nss-tools", Required: true},
}

var pkcs11ModuleCandidates = []string{
	"/usr/local/lib/elo-pkcs11/*.so",
	"/usr/local/lib/advdigital-pkcs11/*.so",
	"/usr/lib/libaetpkss.so",
	"/usr/local/lib/libaetpkss.so",
	"/usr/lib/pkcs11/libaetpkss.so",
	"/usr/lib/libeToken.so",
	"/usr/local/lib/libeToken.so",
	"/usr/lib/pkcs11/libeToken.so",
	"/usr/lib/libIDPrimePKCS11.so",
	"/usr/local/lib/libIDPrimePKCS11.so",
	"/usr/lib/pkcs11/libIDPrimePKCS11.so",
	"/usr/lib64/libaetpkss.so",
	"/usr/lib64/pkcs11/libaetpkss.so",
	"/usr/lib64/libeToken.so",
	"/usr/lib64/pkcs11/libeToken.so",
	"/usr/lib64/libIDPrimePKCS11.so",
	"/usr/lib64/pkcs11/libIDPrimePKCS11.so",
	"/usr/lib64/opensc-pkcs11.so",
	"/usr/lib64/pkcs11/opensc-pkcs11.so",
	"/usr/lib/opensc-pkcs11.so",
	"/usr/lib/pkcs11/opensc-pkcs11.so",
	"/usr/lib/x86_64-linux-gnu/opensc-pkcs11.so",
	"/usr/lib/aarch64-linux-gnu/opensc-pkcs11.so",
}

// App is bound to the Wails frontend.
type App struct {
	ctx context.Context
}

// StatusItem describes one dependency shown in the dashboard.
type StatusItem struct {
	Name        string `json:"name"`
	OK          bool   `json:"ok"`
	Detail      string `json:"detail"`
	Remediation string `json:"remediation"`
}

// SystemStatus is the complete diagnostic payload consumed by Vue.
type SystemStatus struct {
	Java8           StatusItem   `json:"java8"`
	TokenService    StatusItem   `json:"tokenService"`
	TokenReader     StatusItem   `json:"tokenReader"`
	Certificate     StatusItem   `json:"certificate"`
	BrowserPKCS11   StatusItem   `json:"browserPkcs11"`
	Packages        []StatusItem `json:"packages"`
	AllOK           bool         `json:"allOk"`
	CheckedAt       string       `json:"checkedAt"`
	PlatformName    string       `json:"platformName"`
	PackageManager  string       `json:"packageManager"`
	PlatformWarning string       `json:"platformWarning,omitempty"`
}

// ActionResult gives the frontend a stable success/message shape.
type ActionResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved so runtime
// dialogs can be opened by backend methods.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// CheckStatus verifies the Java 8 runtime, smartcard service, token visibility
// and distro-specific packages.
func (a *App) CheckStatus() SystemStatus {
	support := detectLinuxSupport()
	readerOK, readerDetail := smartCardReaderStatus()
	certOK, certDetail := certificateStatus()
	browserOK, browserDetail := browserPKCS11Status()
	javaHome, javaOK := java8Home()
	status := SystemStatus{
		Java8: StatusItem{
			Name:        "Java do tribunal",
			OK:          javaOK,
			Detail:      fmt.Sprintf("Precisa instalar o Java compatível com PJe/Projudi em %s", javaHome),
			Remediation: "Clique em Corrigir Problemas para instalar o Java necessário",
		},
		TokenService: StatusItem{
			Name:        "Leitor do token",
			OK:          isPCSCActive(),
			Detail:      "O computador ainda não está conversando com tokens A3",
			Remediation: "Clique em Corrigir Problemas para ativar o leitor",
		},
		TokenReader: StatusItem{
			Name:        "Token conectado",
			OK:          readerOK,
			Detail:      humanizeTokenDetail(readerDetail),
			Remediation: tokenReaderRemediation(readerDetail),
		},
		Certificate: StatusItem{
			Name:        "Certificado pessoal",
			OK:          certOK,
			Detail:      humanizeCertificateDetail(certDetail),
			Remediation: "Repare ou reemita o certificado no gerenciador do token",
		},
		BrowserPKCS11: StatusItem{
			Name:        "Navegador configurado",
			OK:          browserOK,
			Detail:      humanizeBrowserDetail(browserDetail),
			Remediation: "Clique em Corrigir Problemas e reinicie o navegador",
		},
		CheckedAt:      time.Now().Format("02/01/2006 15:04:05"),
		PlatformName:   support.displayName(),
		PackageManager: support.PackageManager,
	}

	if status.Java8.OK {
		status.Java8.Detail = "Pronto para abrir assinadores do PJe/Projudi"
	}
	if status.TokenService.OK {
		status.TokenService.Detail = "Comunicação com tokens A3 ativada"
	}

	status.Packages = make([]StatusItem, 0, len(requiredPackages))
	for _, pkg := range requiredPackages {
		distroPackage := pkg.nameFor(support)
		ok := isPackageInstalled(support, distroPackage)
		item := StatusItem{
			Name:        pkg.Label,
			OK:          ok,
			Detail:      packageDetail(support, distroPackage),
			Remediation: "Instalar dependências de smartcard",
		}
		if ok {
			item.Detail = "Instalado"
		}
		status.Packages = append(status.Packages, item)
	}

	status.AllOK = status.Java8.OK && status.TokenService.OK && status.TokenReader.OK && status.Certificate.OK && status.BrowserPKCS11.OK
	for i, pkg := range status.Packages {
		if requiredPackages[i].Required {
			status.AllOK = status.AllOK && pkg.OK
		}
	}

	if runtime.GOOS != "linux" {
		status.PlatformWarning = "Este gerenciador foi desenhado para Linux."
	} else if !support.Supported {
		status.PlatformWarning = "Distribuição detectada, mas ainda sem automação completa: " + support.displayName()
	}

	return status
}

// AutoFix installs the required packages and enables pcscd.socket using Polkit.
// pkexec keeps the Wails process unprivileged and asks for the admin password
// through the desktop authentication agent.
func (a *App) AutoFix() (ActionResult, error) {
	if runtime.GOOS != "linux" {
		return ActionResult{OK: false, Message: "AutoFix está disponível apenas no Linux."}, nil
	}
	if _, err := exec.LookPath("pkexec"); err != nil {
		return ActionResult{OK: false, Message: "pkexec não foi encontrado. Instale/ative o Polkit no sistema."}, nil
	}
	support := detectLinuxSupport()
	if !support.Supported {
		return ActionResult{
			OK:      false,
			Message: "Ainda não há correção automática para " + support.displayName() + ". O Elo já automatiza Arch/CachyOS, Fedora e Debian/Ubuntu/Deepin.",
		}, nil
	}

	script := autoFixScript(support)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pkexec", "bash", "-lc", script)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return ActionResult{OK: false, Message: "A correção demorou demais e foi cancelada."}, nil
	}
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		return ActionResult{OK: false, Message: "Não foi possível corrigir automaticamente: " + msg}, nil
	}

	if err := registerBrowserPKCS11(); err != nil {
		return ActionResult{OK: false, Message: "Configuração básica concluída, mas falhou ao configurar o navegador: " + err.Error()}, nil
	}

	status := a.CheckStatus()
	if !status.Java8.OK {
		return ActionResult{
			OK:      false,
			Message: "A configuração básica foi concluída, mas o Java 8 não ficou disponível nesta distribuição. Pode ser necessário instalar um pacote Java 8 compatível manualmente.",
		}, nil
	}
	if !status.TokenReader.OK && needsManufacturerDriver(status.TokenReader.Detail) {
		return ActionResult{
			OK:      false,
			Message: "A configuração básica foi concluída. Este token está presente, mas precisa do driver PKCS#11 do fabricante, como SafeSign Identity Client.",
		}, nil
	}
	if !status.TokenReader.OK {
		return ActionResult{
			OK:      false,
			Message: "A configuração básica foi concluída, mas o token ainda não está pronto: " + status.TokenReader.Detail,
		}, nil
	}

	return ActionResult{OK: true, Message: "Dependências, permissões USB, serviço e token foram configurados."}, nil
}

// SelectProcessFile opens a native picker for .jnlp and .jar files.
func (a *App) SelectProcessFile() (string, error) {
	if a.ctx == nil {
		return "", errors.New("aplicação ainda não inicializada")
	}

	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Selecionar arquivo do tribunal",
		Filters: []wailsruntime.FileFilter{
			{
				DisplayName: "Arquivos do PJe/Projudi (*.jnlp, *.jar)",
				Pattern:     "*.jnlp;*.jar",
			},
		},
	})
}

// SelectDriverFile opens a native picker for proprietary token driver files.
func (a *App) SelectDriverFile() (string, error) {
	if a.ctx == nil {
		return "", errors.New("aplicação ainda não inicializada")
	}

	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Selecionar driver do token",
		Filters: []wailsruntime.FileFilter{
			{
				DisplayName: "Drivers Linux (*.deb, *.rpm, *.tar.gz, *.tgz, *.tar.xz, *.so)",
				Pattern:     "*.deb;*.rpm;*.tar.gz;*.tgz;*.tar.xz;*.so",
			},
		},
	})
}

// InstallDriverFile installs a user-supplied proprietary PKCS#11 driver without
// storing the original installer in the application. The source file must be
// obtained by the user from the token vendor or certificate authority.
func (a *App) InstallDriverFile(path string) (ActionResult, error) {
	cleanPath, kind, err := validateDriverFile(path)
	if err != nil {
		return ActionResult{OK: false, Message: err.Error()}, nil
	}
	if runtime.GOOS != "linux" {
		return ActionResult{OK: false, Message: "Instalação de driver está disponível apenas no Linux."}, nil
	}
	if _, err := exec.LookPath("pkexec"); err != nil {
		return ActionResult{OK: false, Message: "pkexec não foi encontrado. Instale/ative o Polkit no sistema."}, nil
	}

	support := detectLinuxSupport()
	if !support.Supported {
		return ActionResult{
			OK:      false,
			Message: "Ainda não há instalação automática de driver para " + support.displayName() + ".",
		}, nil
	}

	script := driverInstallScript(support, cleanPath, kind)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pkexec", "bash", "-lc", script)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return ActionResult{OK: false, Message: "A instalação do driver demorou demais e foi cancelada."}, nil
	}
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		return ActionResult{OK: false, Message: "Não foi possível instalar o driver: " + msg}, nil
	}

	if err := registerBrowserPKCS11(); err != nil {
		return ActionResult{OK: false, Message: "Driver instalado, mas falhou ao configurar o navegador: " + err.Error()}, nil
	}

	if ok, detail := smartCardReaderStatus(); ok {
		return ActionResult{OK: true, Message: "Driver instalado e certificado detectado: " + detail}, nil
	}

	return ActionResult{
		OK:      false,
		Message: "Driver instalado, mas o certificado ainda não apareceu. Reconecte o token e clique em Atualizar.",
	}, nil
}

// LaunchProcessFile starts the selected .jnlp or .jar with Java 8 isolated in
// the child process environment. The desktop app does not wait for it to exit.
func (a *App) LaunchProcessFile(path string) (ActionResult, error) {
	cleanPath, err := validateProcessFile(path)
	if err != nil {
		return ActionResult{OK: false, Message: err.Error()}, nil
	}
	javaHome, ok := java8Home()
	if !ok {
		return ActionResult{OK: false, Message: "Java 8 não encontrado. Use Corrigir Problemas antes de executar."}, nil
	}

	var cmd *exec.Cmd
	ext := strings.ToLower(filepath.Ext(cleanPath))
	switch ext {
	case ".jnlp":
		javaws, err := findJavaWS(javaHome)
		if err != nil {
			return ActionResult{OK: false, Message: "icedtea-web/javaws não encontrado. Use Corrigir Problemas."}, nil
		}
		cmd = exec.Command(javaws, cleanPath)
	case ".jar":
		javaBin := filepath.Join(javaHome, "bin", "java")
		cmd = exec.Command(javaBin, "-jar", cleanPath)
	default:
		return ActionResult{OK: false, Message: "Use apenas arquivos .jnlp ou .jar."}, nil
	}

	cmd.Dir = filepath.Dir(cleanPath)
	cmd.Env = isolatedJavaEnv(javaHome)

	if err := cmd.Start(); err != nil {
		return ActionResult{OK: false, Message: "Falha ao iniciar o arquivo: " + err.Error()}, nil
	}
	if err := cmd.Process.Release(); err != nil {
		return ActionResult{OK: false, Message: "Processo iniciado, mas não foi possível desacoplar: " + err.Error()}, nil
	}

	return ActionResult{OK: true, Message: "Processo iniciado com Java 8 isolado."}, nil
}

func validateProcessFile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("selecione um arquivo .jnlp ou .jar")
	}

	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return "", errors.New("caminho do arquivo inválido")
	}

	ext := strings.ToLower(filepath.Ext(cleanPath))
	if ext != ".jnlp" && ext != ".jar" {
		return "", errors.New("arquivo inválido. Use apenas .jnlp ou .jar")
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return "", errors.New("arquivo não encontrado")
	}
	if info.IsDir() {
		return "", errors.New("o caminho selecionado é uma pasta")
	}

	return cleanPath, nil
}

func validateDriverFile(path string) (string, string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", "", errors.New("selecione o instalador do driver")
	}

	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return "", "", errors.New("caminho do driver inválido")
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return "", "", errors.New("arquivo do driver não encontrado")
	}
	if info.IsDir() {
		return "", "", errors.New("selecione um arquivo de driver, não uma pasta")
	}

	lower := strings.ToLower(cleanPath)
	switch {
	case strings.HasSuffix(lower, ".deb"):
		return cleanPath, "deb", nil
	case strings.HasSuffix(lower, ".rpm"):
		return cleanPath, "rpm", nil
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"), strings.HasSuffix(lower, ".tar.xz"):
		return cleanPath, "tar", nil
	case strings.HasSuffix(lower, ".so"):
		return cleanPath, "so", nil
	default:
		return "", "", errors.New("formato não suportado. Use .deb, .rpm, .tar.gz, .tgz, .tar.xz ou .so")
	}
}

func detectLinuxSupport() linuxSupport {
	return linuxSupportFromOSRelease(runtime.GOOS, osReleaseValues())
}

func linuxSupportFromOSRelease(goos string, values map[string]string) linuxSupport {
	support := linuxSupport{
		ID:             goos,
		Name:           goos,
		Family:         goos,
		PackageManager: "",
		Supported:      false,
	}
	if goos != "linux" {
		return support
	}

	id := strings.ToLower(values["ID"])
	name := values["PRETTY_NAME"]
	if name == "" {
		name = id
	}
	like := strings.ToLower(values["ID_LIKE"])
	tokens := append(strings.Fields(like), id)

	support.ID = id
	support.Name = name

	if hasAny(tokens, "arch", "cachyos", "manjaro", "endeavouros", "garuda") {
		support.Family = "Arch"
		support.PackageManager = "pacman"
		support.Supported = true
		return support
	}
	if hasAny(tokens, "debian", "ubuntu", "deepin", "linuxmint", "pop", "zorin") {
		support.Family = "Debian/Ubuntu"
		support.PackageManager = "apt"
		support.Supported = true
		return support
	}
	if hasAny(tokens, "fedora") {
		support.Family = "Fedora"
		support.PackageManager = "dnf"
		support.Supported = true
		return support
	}

	support.Family = name
	return support
}

func osReleaseValues() map[string]string {
	content, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return map[string]string{}
	}

	values := map[string]string{}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.Trim(value, `"`)
		values[key] = value
	}
	return values
}

func hasAny(values []string, candidates ...string) bool {
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		for _, candidate := range candidates {
			if value == candidate {
				return true
			}
		}
	}
	return false
}

func (support linuxSupport) displayName() string {
	if support.Name != "" {
		return support.Name
	}
	if support.ID != "" {
		return support.ID
	}
	return "Linux"
}

func autoFixScript(support linuxSupport) string {
	lines := []string{"set -e"}
	lines = append(lines, packageInstallCommands(support, autoFixPackages(support))...)
	lines = append(lines, java8InstallFallbackCommands(support)...)
	lines = append(lines,
		"getent group pcscd >/dev/null || groupadd --system pcscd",
		fmt.Sprintf("printf '%%s\\n' %s > %s", shellQuote(starSignTokenUdevRule), shellQuote(eloUdevRule)),
		"udevadm control --reload-rules",
		"udevadm trigger --subsystem-match=usb || true",
		"systemctl enable --now pcscd.socket || systemctl enable --now pcscd.service || true",
		"systemctl restart pcscd.socket || true",
		"systemctl restart pcscd.service || true",
	)
	return strings.Join(lines, "\n")
}

func autoFixPackages(support linuxSupport) []string {
	packages := requiredPackageNames(support)
	switch support.PackageManager {
	case "pacman":
		return append([]string{"jre8-openjdk", "icedtea-web"}, packages...)
	case "apt":
		return append([]string{"icedtea-netx"}, packages...)
	case "dnf":
		return append([]string{"icedtea-web"}, packages...)
	default:
		return packages
	}
}

func driverInstallPackages(support linuxSupport) []string {
	packages := requiredPackageNames(support)
	switch support.PackageManager {
	case "pacman":
		return append([]string{"libarchive"}, packages...)
	case "apt":
		return append([]string{"libarchive-tools"}, packages...)
	case "dnf":
		return append([]string{"libarchive"}, packages...)
	default:
		return packages
	}
}

func requiredPackageNames(support linuxSupport) []string {
	packages := make([]string, 0, len(requiredPackages))
	for _, spec := range requiredPackages {
		name := spec.nameFor(support)
		if name != "" {
			packages = append(packages, name)
		}
	}
	return packages
}

func (spec packageSpec) nameFor(support linuxSupport) string {
	switch support.PackageManager {
	case "apt":
		return spec.Debian
	case "pacman":
		return spec.Arch
	case "dnf":
		return spec.Fedora
	default:
		if spec.Arch != "" {
			return spec.Arch
		}
		return spec.ID
	}
}

func packageInstallCommands(support linuxSupport, packages []string) []string {
	if len(packages) == 0 {
		return nil
	}

	quoted := make([]string, 0, len(packages))
	for _, pkg := range packages {
		quoted = append(quoted, shellQuote(pkg))
	}

	switch support.PackageManager {
	case "pacman":
		return []string{"pacman -S --needed --noconfirm " + strings.Join(quoted, " ")}
	case "apt":
		return []string{
			"export DEBIAN_FRONTEND=noninteractive",
			"apt-get update",
			"apt-get install -y " + strings.Join(quoted, " "),
		}
	case "dnf":
		return []string{"dnf install -y " + strings.Join(quoted, " ")}
	default:
		return nil
	}
}

func java8InstallFallbackCommands(support linuxSupport) []string {
	switch support.PackageManager {
	case "apt":
		return []string{
			"apt-get install -y openjdk-8-jre || apt-get install -y openjdk-8-jre-headless || true",
		}
	case "dnf":
		return []string{
			"if ! dnf install -y java-1.8.0-openjdk && ! dnf install -y java-1.8.0-openjdk-headless; then",
			"cat > /etc/yum.repos.d/elo-adoptium.repo <<'EOF'",
			"[Adoptium]",
			"name=Eclipse Adoptium",
			"baseurl=https://packages.adoptium.net/artifactory/rpm/fedora/$releasever/$basearch",
			"enabled=1",
			"gpgcheck=1",
			"gpgkey=https://packages.adoptium.net/artifactory/api/gpg/key/public",
			"EOF",
			"dnf install -y temurin-8-jre",
			"fi",
		}
	default:
		return nil
	}
}

func extractDebCommands(path string) []string {
	return []string{
		"tmpdir=$(mktemp -d)",
		"trap 'rm -rf \"$tmpdir\"' EXIT",
		fmt.Sprintf("bsdtar -xf %s -C \"$tmpdir\"", shellQuote(path)),
		"data_archive=$(find \"$tmpdir\" -maxdepth 1 -type f -name 'data.tar*' | head -n 1)",
		"test -n \"$data_archive\"",
		"bsdtar -xf \"$data_archive\" -C /",
	}
}

func driverInstallScript(support linuxSupport, path, kind string) string {
	common := []string{"set -e"}
	common = append(common, packageInstallCommands(support, driverInstallPackages(support))...)
	common = append(common, fmt.Sprintf("mkdir -p %s", shellQuote(eloPKCS11Dir)))

	switch kind {
	case "deb":
		if support.PackageManager == "apt" {
			common = append(common, fmt.Sprintf("apt-get install -y %s", shellQuote(path)))
		} else {
			common = append(common, extractDebCommands(path)...)
		}
	case "rpm":
		if support.PackageManager == "dnf" {
			common = append(common, fmt.Sprintf("dnf install -y %s", shellQuote(path)))
		} else {
			common = append(common, fmt.Sprintf("bsdtar -xf %s -C /", shellQuote(path)))
		}
	case "tar":
		common = append(common, fmt.Sprintf("bsdtar -xf %s -C /", shellQuote(path)))
	case "so":
		common = append(common, fmt.Sprintf("install -m 0644 %s %s/", shellQuote(path), shellQuote(eloPKCS11Dir)))
	}

	common = append(common,
		"ldconfig",
		"udevadm control --reload-rules || true",
		"udevadm trigger --subsystem-match=usb || true",
		"systemctl restart pcscd.socket || true",
		"systemctl restart pcscd.service || true",
	)

	return strings.Join(common, "\n")
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func java8Home() (string, bool) {
	candidates := []string{
		filepath.Join(javaBase, "jre"),
		javaBase,
		"/usr/lib/jvm/java-8-openjdk-amd64/jre",
		"/usr/lib/jvm/java-8-openjdk-amd64",
		"/usr/lib/jvm/java-8-openjdk-arm64/jre",
		"/usr/lib/jvm/java-8-openjdk-arm64",
		"/usr/lib/jvm/java-1.8.0-openjdk/jre",
		"/usr/lib/jvm/java-1.8.0-openjdk",
		"/usr/lib/jvm/jre-1.8.0-openjdk",
		"/usr/lib/jvm/default-runtime",
	}
	candidates = append(candidates, java8GlobCandidates()...)

	for _, candidate := range candidates {
		javaBin := filepath.Join(candidate, "bin", "java")
		if info, err := os.Stat(javaBin); err == nil && !info.IsDir() && isJava8(javaBin) {
			return candidate, true
		}
	}

	return filepath.Join(javaBase, "jre"), false
}

func java8GlobCandidates() []string {
	patterns := []string{
		"/usr/lib/jvm/*8*",
		"/usr/lib/jvm/*1.8*",
	}
	candidates := []string{}
	seen := map[string]bool{}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			for _, candidate := range []string{filepath.Join(match, "jre"), match} {
				if seen[candidate] {
					continue
				}
				seen[candidate] = true
				candidates = append(candidates, candidate)
			}
		}
	}
	return candidates
}

func isJava8(javaBin string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, javaBin, "-version").CombinedOutput()
	return err == nil && strings.Contains(string(output), `version "1.8.`)
}

func isPackageInstalled(support linuxSupport, pkg string) bool {
	if pkg == "" || !support.Supported {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	switch support.PackageManager {
	case "pacman":
		return exec.CommandContext(ctx, "pacman", "-Q", pkg).Run() == nil
	case "apt":
		output, err := exec.CommandContext(ctx, "dpkg-query", "-W", "-f=${db:Status-Status}", pkg).CombinedOutput()
		return err == nil && strings.TrimSpace(string(output)) == "installed"
	case "dnf":
		return exec.CommandContext(ctx, "rpm", "-q", pkg).Run() == nil
	default:
		return false
	}
}

func packageDetail(support linuxSupport, pkg string) string {
	if !support.Supported {
		return "Distribuição ainda sem verificação automática de pacote"
	}
	switch support.PackageManager {
	case "apt":
		return "Pacote apt: " + pkg
	case "pacman":
		return "Pacote pacman: " + pkg
	case "dnf":
		return "Pacote dnf: " + pkg
	default:
		return "Pacote: " + pkg
	}
}

func isPCSCActive() bool {
	return systemctlIsActive("pcscd.socket") || systemctlIsActive("pcscd.service") || systemctlIsActive("pcscd")
}

func smartCardReaderStatus() (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "opensc-tool", "-l").CombinedOutput()
	text := strings.TrimSpace(string(output))
	if ctx.Err() == context.DeadlineExceeded {
		return false, "Tempo esgotado ao consultar o token"
	}
	if err != nil && text == "" {
		return false, "OpenSC ainda não conseguiu consultar leitores"
	}
	if strings.Contains(text, "No smart card readers found") {
		if usbTokenDetected() {
			return false, "Token USB encontrado, mas sem permissão PC/SC"
		}
		return false, "Nenhum token/leitor conectado"
	}
	if text == "" {
		return false, "Nenhum leitor retornado pelo OpenSC"
	}

	slotOK, slotDetail := pkcs11TokenStatus()
	if !slotOK {
		if needsManufacturerDriver(slotDetail) {
			return false, slotDetail
		}
		reader := firstReaderLine(text)
		if reader != "" {
			return false, "Leitor detectado, mas o token está vazio: " + reader
		}
		return false, slotDetail
	}

	return true, slotDetail
}

func systemctlIsActive(unit string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", unit).Run() == nil
}

func findJavaWS(javaHome string) (string, error) {
	candidates := []string{
		filepath.Join(javaHome, "bin", "javaws"),
		"/usr/bin/javaws",
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return exec.LookPath("javaws")
}

func isolatedJavaEnv(javaHome string) []string {
	env := os.Environ()
	env = upsertEnv(env, "JAVA_HOME", javaHome)
	env = upsertEnv(env, "PATH", filepath.Join(javaHome, "bin")+":"+os.Getenv("PATH"))
	env = upsertEnv(env, "LD_LIBRARY_PATH", libraryPathWith("/usr/lib", os.Getenv("LD_LIBRARY_PATH")))
	env = upsertEnv(env, "JAVA_TOOL_OPTIONS", javaToolOptionsWithPKCS11(os.Getenv("JAVA_TOOL_OPTIONS")))
	return env
}

func libraryPathWith(path, current string) string {
	if current == "" {
		return path
	}
	for _, item := range strings.Split(current, ":") {
		if item == path {
			return current
		}
	}
	return path + ":" + current
}

func javaToolOptionsWithPKCS11(current string) string {
	module, ok := preferredPKCS11Module()
	if !ok {
		return current
	}

	securityFile, err := ensureJavaPKCS11Config(module)
	if err != nil {
		return current
	}

	option := "-Djava.security.properties=" + securityFile
	if strings.Contains(current, option) {
		return current
	}
	if strings.TrimSpace(current) == "" {
		return option
	}
	return current + " " + option
}

func ensureJavaPKCS11Config(module string) (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "elo")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}

	providerPath := filepath.Join(dir, "safesign-pkcs11.cfg")
	securityPath := filepath.Join(dir, "java-security.properties")

	providerConfig := strings.Join([]string{
		"name = EloSafeSign",
		"library = " + module,
		"slotListIndex = 0",
	}, "\n") + "\n"

	securityConfig := "security.provider.10=sun.security.pkcs11.SunPKCS11 " + providerPath + "\n"

	if err := os.WriteFile(providerPath, []byte(providerConfig), 0o600); err != nil {
		return "", err
	}
	if err := os.WriteFile(securityPath, []byte(securityConfig), 0o600); err != nil {
		return "", err
	}

	return securityPath, nil
}

func preferredPKCS11Module() (string, bool) {
	for _, module := range installedPKCS11Modules() {
		if strings.Contains(filepath.Base(module), "aetpkss") {
			return module, true
		}
	}
	modules := installedPKCS11Modules()
	if len(modules) > 0 {
		return modules[0], true
	}
	return "", false
}

func usbTokenDetected() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "lsusb", "-d", "1059:0019").CombinedOutput()
	return err == nil && strings.TrimSpace(string(output)) != ""
}

func firstUsefulLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			return line
		}
	}
	return "Leitor de smartcard detectado"
}

func firstReaderLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "Nr.") || strings.Contains(line, "Card  Features") {
			continue
		}
		return line
	}
	return ""
}

func pkcs11TokenStatus() (bool, string) {
	modules := installedPKCS11Modules()

	var lastDetail string
	for _, module := range modules {
		ok, detail := pkcs11TokenStatusForModule(module)
		if ok {
			return true, detail
		}
		if detail != "" {
			lastDetail = detail
		}
	}

	if lastDetail != "" {
		return false, lastDetail
	}
	return false, "Nenhum módulo PKCS#11 conseguiu ler o token"
}

func pkcs11TokenStatusForModule(module string) (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "pkcs11-tool", "--module", module, "-L").CombinedOutput()
	text := strings.TrimSpace(string(output))
	if ctx.Err() == context.DeadlineExceeded {
		return false, "Tempo esgotado ao consultar o slot do token"
	}
	if err != nil && text == "" {
		return false, "PKCS#11 ainda não encontrou o token"
	}
	if strings.Contains(text, "token not recognized") {
		return false, "Cartão presente, mas o modelo não foi reconhecido pelo OpenSC"
	}
	hasTokenMetadata := strings.Contains(text, "token label") ||
		strings.Contains(text, "token manufacturer") ||
		strings.Contains(text, "token model")
	if !hasTokenMetadata && (strings.Contains(text, "(empty)") || strings.Contains(text, "No slot with a token")) {
		return false, "Leitor encontrado, mas sem token/certificado disponível"
	}

	objectsOK, objectsDetail := pkcs11ObjectsStatusForModule(module)
	if !objectsOK {
		return false, objectsDetail
	}

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "token label") {
			return true, fmt.Sprintf("%s (%s)", line, filepath.Base(module))
		}
	}

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "manufacturer") ||
			strings.HasPrefix(line, "Slot ") {
			return true, fmt.Sprintf("%s (%s)", line, filepath.Base(module))
		}
	}

	if text != "" {
		return true, firstUsefulLine(text)
	}
	return false, "Nenhum slot PKCS#11 disponível"
}

func pkcs11ObjectsStatus() (bool, string) {
	for _, module := range installedPKCS11Modules() {
		if strings.Contains(filepath.Base(module), "opensc-pkcs11") {
			return pkcs11ObjectsStatusForModule(module)
		}
	}
	return false, "Módulo PKCS#11 do OpenSC não encontrado"
}

func pkcs11ObjectsStatusForModule(module string) (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "pkcs11-tool", "--module", module, "-O").CombinedOutput()
	text := strings.TrimSpace(string(output))
	if ctx.Err() == context.DeadlineExceeded {
		return false, "Tempo esgotado ao consultar certificados do token"
	}
	if strings.Contains(text, "token not recognized") {
		return false, "Cartão presente, mas precisa de driver PKCS#11 do fabricante"
	}
	if strings.Contains(text, "No slot with a token") {
		return false, "Leitor encontrado, mas sem token/certificado disponível"
	}
	if err != nil && text == "" {
		return false, "Não foi possível listar certificados do token"
	}
	if strings.Contains(text, "Certificate Object") || strings.Contains(text, "Private Key Object") {
		return true, firstTokenObjectLine(text)
	}
	if strings.Contains(text, "Using slot") {
		return false, "Token presente, mas nenhum certificado foi listado"
	}
	if text != "" {
		return true, firstUsefulLine(text)
	}
	return false, "Nenhum certificado encontrado no token"
}

func installedPKCS11Modules() []string {
	seen := map[string]bool{}
	modules := []string{}
	for _, pattern := range pkcs11ModuleCandidates {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if seen[match] {
				continue
			}
			info, err := os.Stat(match)
			if err == nil && !info.IsDir() {
				seen[match] = true
				modules = append(modules, match)
			}
		}
	}
	return modules
}

func browserPKCS11Status() (bool, string) {
	module, ok := preferredPKCS11Module()
	if !ok {
		return false, "Driver PKCS#11 do token ainda não foi encontrado"
	}

	dbs := browserNSSDatabases()
	if len(dbs) == 0 {
		return false, "Banco de certificados do navegador não encontrado"
	}

	configured := 0
	for _, db := range dbs {
		if nssDatabaseHasModule(db, module) {
			configured++
		}
	}
	if configured == 0 {
		return false, "Driver do token ainda não registrado no navegador"
	}
	if configured < len(dbs) {
		return false, fmt.Sprintf("Driver registrado em %d de %d perfis de navegador", configured, len(dbs))
	}

	return true, fmt.Sprintf("Driver registrado no navegador (%s)", filepath.Base(module))
}

type tokenCertificate struct {
	Label    string
	ID       string
	NotAfter time.Time
}

func certificateStatus() (bool, string) {
	module, ok := preferredPKCS11Module()
	if !ok {
		return false, "Driver PKCS#11 do token não encontrado"
	}

	certs := tokenCertificates(module)
	if len(certs) == 0 {
		return false, "Nenhum certificado encontrado no token"
	}

	publicKeyIDs := tokenPublicKeyIDs(module)
	now := time.Now()
	var newestValid *tokenCertificate
	var expiredWithKey *tokenCertificate

	for i := range certs {
		cert := &certs[i]
		if cert.ID == "" || cert.NotAfter.IsZero() {
			continue
		}
		if cert.NotAfter.Before(now) {
			if publicKeyIDs[cert.ID] {
				expiredWithKey = cert
			}
			continue
		}
		if newestValid == nil || cert.NotAfter.After(newestValid.NotAfter) {
			newestValid = cert
		}
		if publicKeyIDs[cert.ID] {
			return true, fmt.Sprintf("%s válido até %s", cert.Label, cert.NotAfter.Format("02/01/2006"))
		}
	}

	if newestValid != nil {
		return false, fmt.Sprintf("%s está válido até %s, mas não tem chave associada visível", newestValid.Label, newestValid.NotAfter.Format("02/01/2006"))
	}
	if expiredWithKey != nil {
		return false, fmt.Sprintf("%s tem chave associada, mas venceu em %s", expiredWithKey.Label, expiredWithKey.NotAfter.Format("02/01/2006"))
	}

	return false, "Nenhum certificado pessoal válido com chave associada foi encontrado"
}

func tokenCertificates(module string) []tokenCertificate {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "pkcs11-tool", "--module", module, "-O", "--type", "cert").CombinedOutput()
	if err != nil && strings.TrimSpace(string(output)) == "" {
		return nil
	}

	blocks := splitPKCS11ObjectBlocks(string(output), "Certificate Object")
	certs := make([]tokenCertificate, 0, len(blocks))
	for _, block := range blocks {
		label := fieldFromPKCS11Block(block, "label")
		id := normalizePKCS11ID(multilineFieldFromPKCS11Block(block, "ID"))
		if label == "" || id == "" || strings.HasPrefix(label, "AC ") || strings.Contains(label, "Autoridade Certificadora") {
			continue
		}
		notAfter := certificateExpiry(module, id)
		certs = append(certs, tokenCertificate{Label: label, ID: id, NotAfter: notAfter})
	}
	return certs
}

func tokenPublicKeyIDs(module string) map[string]bool {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "pkcs11-tool", "--module", module, "-O", "--type", "pubkey").CombinedOutput()
	if err != nil && strings.TrimSpace(string(output)) == "" {
		return nil
	}

	ids := map[string]bool{}
	for _, block := range splitPKCS11ObjectBlocks(string(output), "Public Key Object") {
		id := normalizePKCS11ID(multilineFieldFromPKCS11Block(block, "ID"))
		if id != "" {
			ids[id] = true
		}
	}
	return ids
}

func certificateExpiry(module, id string) time.Time {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	readCmd := exec.CommandContext(ctx, "pkcs11-tool", "--module", module, "--read-object", "--type", "cert", "--id", id)
	opensslCmd := exec.CommandContext(ctx, "openssl", "x509", "-inform", "DER", "-noout", "-enddate")

	reader, err := readCmd.StdoutPipe()
	if err != nil {
		return time.Time{}
	}
	opensslCmd.Stdin = reader
	if err := readCmd.Start(); err != nil {
		return time.Time{}
	}
	output, err := opensslCmd.Output()
	_ = readCmd.Wait()
	if err != nil {
		return time.Time{}
	}

	text := strings.TrimSpace(string(output))
	text = strings.TrimPrefix(text, "notAfter=")
	expires, err := time.Parse("Jan 2 15:04:05 2006 MST", text)
	if err != nil {
		return time.Time{}
	}
	return expires
}

func splitPKCS11ObjectBlocks(text, marker string) []string {
	parts := strings.Split(text, marker)
	blocks := []string{}
	for _, part := range parts[1:] {
		part = strings.TrimSpace(marker + part)
		if part != "" {
			blocks = append(blocks, part)
		}
	}
	return blocks
}

func fieldFromPKCS11Block(block, field string) string {
	prefix := field + ":"
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func multilineFieldFromPKCS11Block(block, field string) string {
	prefix := field + ":"
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		for _, next := range lines[i+1:] {
			if !strings.HasPrefix(next, " ") && !strings.HasPrefix(next, "\t") {
				break
			}
			next = strings.TrimSpace(next)
			if isPKCS11IDLine(next) {
				value += ":" + next
				continue
			}
			break
		}
		return value
	}
	return ""
}

func isPKCS11IDLine(line string) bool {
	if line == "" {
		return false
	}
	for _, part := range strings.Split(line, ":") {
		if len(part) != 2 {
			return false
		}
		for _, r := range part {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return false
			}
		}
	}
	return true
}

func normalizePKCS11ID(id string) string {
	replacer := strings.NewReplacer(":", "", " ", "", "\t", "", "\n", "")
	return strings.ToLower(replacer.Replace(id))
}

func registerBrowserPKCS11() error {
	module, ok := preferredPKCS11Module()
	if !ok {
		return errors.New("driver PKCS#11 do token não encontrado")
	}
	if _, err := exec.LookPath("modutil"); err != nil {
		return errors.New("modutil não encontrado. Instale o pacote nss")
	}
	if _, err := exec.LookPath("certutil"); err != nil {
		return errors.New("certutil não encontrado. Instale o pacote nss")
	}

	if err := ensureChromiumNSSDatabase(); err != nil {
		return err
	}

	dbs := browserNSSDatabases()
	if len(dbs) == 0 {
		return errors.New("nenhum perfil de navegador encontrado")
	}

	var failures []string
	for _, db := range dbs {
		if nssDatabaseHasModule(db, module) {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		output, err := exec.CommandContext(ctx,
			"modutil",
			"-force",
			"-dbdir", "sql:"+db,
			"-add", "Elo Token A3",
			"-libfile", module,
		).CombinedOutput()
		cancel()
		if err != nil && !strings.Contains(string(output), "already exists") {
			failures = append(failures, filepath.Base(db)+": "+strings.TrimSpace(string(output)))
		}
	}

	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "; "))
	}
	return nil
}

func ensureChromiumNSSDatabase() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	db := filepath.Join(home, ".pki", "nssdb")
	if _, err := os.Stat(filepath.Join(db, "cert9.db")); err == nil {
		return nil
	}
	if err := os.MkdirAll(db, 0o700); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "certutil", "-N", "--empty-password", "-d", "sql:"+db).CombinedOutput()
	if err != nil {
		return errors.New(strings.TrimSpace(string(output)))
	}
	return nil
}

func browserNSSDatabases() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	dbs := []string{}

	candidates := []string{filepath.Join(home, ".pki", "nssdb")}
	firefoxMatches, _ := filepath.Glob(filepath.Join(home, ".mozilla", "firefox", "*", "cert9.db"))
	for _, match := range firefoxMatches {
		candidates = append(candidates, filepath.Dir(match))
	}

	for _, candidate := range candidates {
		if seen[candidate] {
			continue
		}
		if info, err := os.Stat(filepath.Join(candidate, "cert9.db")); err == nil && !info.IsDir() {
			seen[candidate] = true
			dbs = append(dbs, candidate)
		}
	}
	return dbs
}

func nssDatabaseHasModule(db, module string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "modutil", "-dbdir", "sql:"+db, "-list").CombinedOutput()
	if err != nil {
		return false
	}

	text := string(output)
	return strings.Contains(text, module) ||
		strings.Contains(text, "Elo Token A3") ||
		strings.Contains(text, "AdvDigital Token A3") ||
		strings.Contains(text, filepath.Base(module))
}

func needsManufacturerDriver(detail string) bool {
	detail = strings.ToLower(detail)
	return strings.Contains(detail, "não foi reconhecido") ||
		strings.Contains(detail, "driver pkcs#11") ||
		strings.Contains(detail, "fabricante")
}

func humanizeTokenDetail(detail string) string {
	lower := strings.ToLower(detail)
	switch {
	case strings.Contains(lower, "token usb encontrado"):
		return "Token USB encontrado, mas o Linux ainda não tem permissão para acessá-lo"
	case strings.Contains(lower, "não foi reconhecido") || strings.Contains(lower, "driver pkcs#11"):
		return "Token presente, mas precisa do driver correto do fabricante"
	case strings.Contains(lower, "sem token") || strings.Contains(lower, "vazio"):
		return "Leitor encontrado, mas nenhum token ativo foi lido"
	case strings.Contains(lower, "nenhum token") || strings.Contains(lower, "nenhum leitor"):
		return "Conecte o token A3 e clique em Atualizar"
	}

	if label := tokenLabelFromPKCS11Detail(detail); label != "" {
		return "Token " + label + " conectado"
	}
	if strings.Contains(detail, "libaetpkss.so") {
		return "Token conectado com driver SafeSign"
	}
	if strings.Contains(detail, "Slot ") {
		return "Token conectado e respondendo"
	}
	return detail
}

func tokenLabelFromPKCS11Detail(detail string) string {
	for _, line := range strings.Split(detail, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(strings.ToLower(line), "token label") {
			continue
		}
		_, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.TrimSuffix(value, "(libaetpkss.so)")
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func humanizeCertificateDetail(detail string) string {
	if strings.Contains(detail, " válido até ") {
		label, expires, _ := strings.Cut(detail, " válido até ")
		return fmt.Sprintf("Certificado de %s válido até %s", sanitizeCertificateLabel(label), expires)
	}
	if strings.Contains(detail, " está válido até ") {
		label, rest, _ := strings.Cut(detail, " está válido até ")
		return fmt.Sprintf("Certificado de %s está válido até %s", sanitizeCertificateLabel(label), rest)
	}
	if strings.Contains(detail, " tem chave associada") {
		label, rest, _ := strings.Cut(detail, " tem chave associada")
		return fmt.Sprintf("Certificado de %s tem chave associada%s", sanitizeCertificateLabel(label), rest)
	}
	return detail
}

func sanitizeCertificateLabel(label string) string {
	label = strings.TrimSpace(label)
	if before, after, ok := strings.Cut(label, ":"); ok && looksNumeric(after) {
		return strings.TrimSpace(before)
	}
	return label
}

func looksNumeric(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func humanizeBrowserDetail(detail string) string {
	lower := strings.ToLower(detail)
	switch {
	case strings.Contains(lower, "driver registrado no navegador"):
		return "Navegador pronto para acessar o token"
	case strings.Contains(lower, "registrado em"):
		return "Parte dos perfis do navegador já foi configurada"
	case strings.Contains(lower, "banco de certificados"):
		return "Abra o navegador uma vez e clique em Corrigir Problemas"
	case strings.Contains(lower, "driver pkcs#11"):
		return "Instale o driver do token para configurar o navegador"
	case strings.Contains(lower, "ainda não registrado"):
		return "O navegador ainda precisa receber o driver do token"
	default:
		return detail
	}
}

func tokenReaderRemediation(detail string) string {
	if needsManufacturerDriver(detail) {
		return "Instale o driver do fabricante pelo botão Instalar Driver do Token"
	}
	if strings.Contains(strings.ToLower(detail), "sem token") ||
		strings.Contains(strings.ToLower(detail), "vazio") {
		return "Reconecte o token e clique em Atualizar"
	}
	return "Conecte o token ou clique em Corrigir Problemas"
}

func firstTokenObjectLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Certificate Object") || strings.Contains(line, "Private Key Object") {
			return line
		}
	}
	return "Certificado encontrado no token"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func upsertEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, item := range env {
		if strings.HasPrefix(item, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
