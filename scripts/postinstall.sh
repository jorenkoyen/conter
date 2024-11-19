#!/bin/bash

# styling
BOLD="$(tput bold 2>/dev/null || printf '')"
GREY="$(tput setaf 0 2>/dev/null || printf '')"
UNDERLINE="$(tput smul 2>/dev/null || printf '')"
RED="$(tput setaf 1 2>/dev/null || printf '')"
GREEN="$(tput setaf 2 2>/dev/null || printf '')"
YELLOW="$(tput setaf 3 2>/dev/null || printf '')"
BLUE="$(tput setaf 4 2>/dev/null || printf '')"
NO_COLOR="$(tput sgr0 2>/dev/null || printf '')"

# default configuration variables
USER="conter"
GROUP="conter"
CONFIG_DIR="/etc/conter"
DATA_DIR="/var/lib/conter"
CONFIG_FILE="$CONFIG_DIR/config.toml"
SYSTEMD_FILE="/etc/systemd/system/conter.service"
SYSTEMD_SERVICE="conter.service"
BINARY_PATH=$(which conter)

info() {
  printf '%s\n' "${BOLD}${GREY}>${NO_COLOR} $*"
}

warn() {
  printf '%s\n' "${YELLOW}! $*${NO_COLOR}"
}

error() {
  printf '%s\n' "${RED}x $*${NO_COLOR}" >&2
}

completed() {
  printf '%s\n' "${GREEN}✓${NO_COLOR} $*"
}


# 1. Create system user and group
if ! id -u "$USER" >/dev/null 2>&1; then
    info "Creating system user and group: ${BLUE}${USER}${NO_COLOR}:${BLUE}${GROUP}${NO_COLOR}"
    useradd --system --no-create-home --shell /usr/sbin/nologin "$USER"
fi

# 2. Create configuration directory
if [ ! -d "$CONFIG_DIR" ]; then
    info "Creating configuration directory: ${BLUE}${CONFIG_DIR}${NO_COLOR}"
    mkdir -p "$CONFIG_DIR"
    chown "$USER:$GROUP" "$CONFIG_DIR"
    chmod 750 "$CONFIG_DIR"
fi

# 3. Create data directory
if [ ! -d "$DATA_DIR" ]; then
    info "Creating data directory: ${BLUE}${DATA_DIR}${NO_COLOR}"
    mkdir -p "$DATA_DIR"
    chown "$USER:$GROUP" "$DATA_DIR"
    chmod 750 "$DATA_DIR"
fi

# 4. Generate configuration file if not already present
if [ ! -f "$CONFIG_FILE" ]; then
    cat <<EOF > "$CONFIG_FILE"
log_level       = "info"
listen_address  = "127.0.0.1:6440"

[acme]
email     = ""
directory = "https://acme-staging-v02.api.letsencrypt.org/directory"
insecure  = false

[data]
directory = "$DATA_DIR"

[proxy]
http_listen_address  = "0.0.0.0:80"
https_listen_address = "0.0.0.0:443"
EOF

    chown "$USER:$GROUP" "$CONFIG_FILE"
    chmod 640 "$CONFIG_FILE"
    info "Configuration file created at ${BLUE}${CONFIG_FILE}${NO_COLOR}."
    warn "Please update your ACME email accordingly before using Conter"
else
    warn "Configuration file already exists at $CONFIG_FILE. Skipping creation."
fi

# 5. Generate systemd service file
info "Generating systemd service file at ${BLUE}${SYSTEMD_FILE}${NO_COLOR}"
mkdir -p /etc/systemd/system
cat <<EOF > "$SYSTEMD_FILE"
[Unit]
Description=A minimal container management system for small scale web deployments
After=network.target

[Service]
ExecStartPre=$BINARY_PATH --config $CONFIG_FILE --validate-config
ExecStart=$BINARY_PATH --config $CONFIG_FILE
User=$USER
Group=$GROUP
WorkingDirectory=$DATA_DIR
AmbientCapabilities=CAP_NET_BIND_SERVICE
Restart=always

[Install]
WantedBy=multi-user.target
EOF

# Set permissions for the systemd service file
chmod 644 "$SYSTEMD_FILE"

# Reload systemd to acknowledge the new service file
info "Reloading systemd daemon to apply changes."
systemctl daemon-reload

# enable and start systemd service
info "Enabling and starting ${BLUE}${SYSTEMD_SERVICE}${NO_COLOR} systemd service"
systemctl enable --quiet --now ${SYSTEMD_SERVICE}

version=$(conter --version)
completed "Conter has been successfully installed with version ${version}"
