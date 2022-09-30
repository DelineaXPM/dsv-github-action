# Note: You can use any Debian/Ubuntu based image you want.
FROM mcr.microsoft.com/vscode/devcontainers/universal:2-focal
ENV SHELL=/bin/bash

# Enable new "BUILDKIT" mode for Docker CLI
ENV DOCKER_BUILDKIT=1

# Options
ARG INSTALL_ZSH="true"
ARG UPGRADE_PACKAGES="false"
ARG USE_MOBY="true"
ARG USERNAME=codespace
ARG USER_UID=1000
ARG USER_GID=$USER_UID
ARG ENABLE_NONROOT_DOCKER="true"
ARG DOCKER_VERSION="latest"
ARG GO_VERSION="${GO_VERSION}"
ARG GOPATH="${GOPATH}"
ARG GOROOT="${GO_ROOT}"

# Install needed packages and setup non-root user. Use a separate RUN statement to add your own dependencies.
USER root

ENV GOROOT=${GOROOT}
ENV GOPATH=${GOPATH}
ENV PATH=${GOPATH}:$PATH
ENV MAGEFILE_ENABLE_COLOR=1

COPY library-scripts/*.sh /tmp/library-scripts/
RUN apt-get update && /bin/bash /tmp/library-scripts/common-debian.sh "${INSTALL_ZSH}" "${USERNAME}" "${USER_UID}" "${USER_GID}" "${UPGRADE_PACKAGES}" "true" "true" \
  && /bin/bash /tmp/library-scripts/docker-in-docker-debian.sh "${ENABLE_NONROOT_DOCKER}" "${USERNAME}" "${USE_MOBY}" "${DOCKER_VERSION}" \
  && /bin/bash /tmp/library-scripts/go-debian.sh "${GO_VERSION}" "${GOROOT}" "${GOPATH}" "automatic" "true" "true" \
  && /bin/bash /tmp/library-scripts/update-golang.sh \
  && mkdir -p /home/${USERNAME}/.minikube \
  && chown ${USERNAME} /home/${USERNAME}/.minikube \
  && apt-get autoremove -y && apt-get clean -y && rm -rf /var/lib/apt/lists/* /tmp/library-scripts/
VOLUME [ "/var/lib/docker" ]
ENTRYPOINT ["/usr/local/share/docker-init.sh"]
CMD ["sleep", "infinity"]
RUN /bin/bash -c 'set -euo pipefail && curl -sfL https://direnv.net/install.sh | bash'
USER ${USERNAME}
RUN go install github.com/magefile/mage@latest && go install github.com/charmbracelet/glow@latest
# trunk-ignore(hadolint/DL3013)
RUN python3 -m pip install --upgrade pip --user --no-cache-dir && python3 -m pip install pre-commit --user --no-cache-dir
COPY files/.zshrc /home/${USERNAME}/.zshrc
COPY files/first-run-notice.txt /home/${USERNAME}/first-run-notice.txt
