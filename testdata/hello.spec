Name: hello
Version: 1.0
Release: 1.el9
Summary: Aptify RPM test package
License: MIT
BuildArch: x86_64

%description
A minimal RPM used by Aptify integration tests.

%prep

%build

%install
mkdir -p %{buildroot}/usr/bin
cat > %{buildroot}/usr/bin/hello <<'EOF'
#!/bin/sh
echo "hello from Aptify"
EOF
chmod 0755 %{buildroot}/usr/bin/hello

%files
/usr/bin/hello
