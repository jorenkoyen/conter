#!/bin/bash

# styling
BOLD="$(tput bold 2>/dev/null || printf '')"
GREY="$(tput setaf 0 2>/dev/null || printf '')"
UNDERLINE="$(tput smul 2>/dev/null || printf '')"
NO_UNDERLINE="$(tput rmul 2>/dev/null || printf '')"
RED="$(tput setaf 1 2>/dev/null || printf '')"
GREEN="$(tput setaf 2 2>/dev/null || printf '')"
YELLOW="$(tput setaf 3 2>/dev/null || printf '')"
BLUE="$(tput setaf 4 2>/dev/null || printf '')"
NO_COLOR="$(tput sgr0 2>/dev/null || printf '')"

# default configuration variables
CONFIG_DIR="/etc/conter"
DATA_DIR="/var/lib/conter"
SYSTEMD_SERVICE="conter.service"

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

# stop systemctl daemon
#if systemctl is-active --quiet ${SYSTEMD_SERVICE}; then
#  info "Stopping systemd service ${BLUE}${SYSTEMD_SERVICE}${NO_COLOR}"
#  systemctl stop ${SYSTEMD_SERVICE}
#  systemctl disable ${SYSTEMD_SERVICE}
#fi

# give warning to cleanup directories
warning "Remove ${UNDERLINE}${DATA_DIR}${NO_UNDERLINE} to delete application data"
warning "Remove ${UNDERLINE}${CONFIG_DIR}${NO_UNDERLINE} to delete configuration files"