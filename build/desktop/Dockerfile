FROM golang:1.27-bookworm

# Instala dependências de compilação CGO para Linux (GTK3, WebKit2GTK 4.1),
# Windows cross-compilation (MinGW-w64), ferramentas de conversão de ícones (rsvg-convert, icoutils)
# e utilitários de empacotamento.
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgtk-3-dev \
    libwebkit2gtk-4.1-dev \
    pkg-config \
    gcc \
    gcc-mingw-w64-x86-64 \
    librsvg2-bin \
    icoutils \
    curl \
    ca-certificates \
    rsync \
    file \
    zip \
    && rm -rf /var/lib/apt/lists/*

# Instala Wails CLI v2 oficial
RUN go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Instala appimagetool extraído (executa sem FUSE em containers Docker)
RUN curl -sL https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage -o /tmp/appimagetool.AppImage \
    && chmod +x /tmp/appimagetool.AppImage \
    && cd /tmp && ./appimagetool.AppImage --appimage-extract \
    && mv /tmp/squashfs-root /opt/appimagetool \
    && ln -s /opt/appimagetool/AppRun /usr/local/bin/appimagetool \
    && rm -f /tmp/appimagetool.AppImage

WORKDIR /workspace

CMD ["wails", "version"]
