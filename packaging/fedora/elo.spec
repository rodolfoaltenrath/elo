Name:           elo
Version:        0.1.0
Release:        1%{?dist}
Summary:        Integra Linux ao PJe/Projudi com Java 8 e certificados digitais

License:        LicenseRef-Not-Specified
URL:            https://github.com/rodolfoaltenrath/elo
Source0:        elo
Source1:        elo.desktop
Source2:        elo.png
Source3:        README.md

Requires:       gtk3
Requires:       webkit2gtk4.1
Requires:       nss-tools
Requires:       openssl
Requires:       opensc
Requires:       pcsc-lite
Requires:       pcsc-lite-ccid
Requires:       polkit
Requires:       usbutils
Recommends:     icedtea-web
Recommends:     pcsc-tools

%description
O Elo prepara e diagnostica Java 8, leitores PC/SC, tokens A3, módulos
PKCS#11 e perfis NSS do navegador para uso com PJe e Projudi.

%prep

%build

%install
install -Dm0755 %{SOURCE0} %{buildroot}%{_bindir}/elo
install -Dm0644 %{SOURCE1} %{buildroot}%{_datadir}/applications/elo.desktop
install -Dm0644 %{SOURCE2} %{buildroot}%{_datadir}/icons/hicolor/1024x1024/apps/elo.png
install -Dm0644 %{SOURCE3} %{buildroot}%{_docdir}/elo/README.md

%files
%{_bindir}/elo
%{_datadir}/applications/elo.desktop
%{_datadir}/icons/hicolor/1024x1024/apps/elo.png
%doc %{_docdir}/elo/README.md

%changelog
* Thu Jul 30 2026 Elo maintainers <rodolfoaltenrath@users.noreply.github.com> - 0.1.0-1
- Adiciona pacote inicial para Fedora
