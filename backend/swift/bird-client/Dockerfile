ARG SWIFT_VERSION=6.1
ARG SWIFT_PLATFORM=amazonlinux2-aarch64
ARG SWIFT_BRANCH=swift-${SWIFT_VERSION}-RELEASE
ARG SWIFT_URL=https://download.swift.org/swift-${SWIFT_VERSION}-release/${SWIFT_PLATFORM}/${SWIFT_BRANCH}/${SWIFT_BRANCH}-${SWIFT_PLATFORM}.tar.gz
ARG SWIFT_DIR_NAME=${SWIFT_BRANCH}-${SWIFT_PLATFORM}
ARG SWIFT_INSTALL_DIR_BASE=/opt/swift

FROM public.ecr.aws/amazonlinux/amazonlinux:2

ARG SWIFT_URL
ARG SWIFT_DIR_NAME
ARG SWIFT_INSTALL_DIR_BASE

RUN yum update -y && \
    yum install -y \
    tar \
    gzip \
    curl \
    libuuid-devel \
    libicu-devel \
    libedit-devel \
    libxml2-devel \
    sqlite-devel \
    python3-pip \
    git \
    binutils \
    gcc \
    gzip \
    glibc-static \
    libcurl-devel \
    libstdc++-static \
    zlib-devel \
    which \
    make \
    openssl-devel && \
    yum clean all

RUN pip3 install --upgrade pip && \
    pip3 install aws-lambda-builders


RUN mkdir -p ${SWIFT_INSTALL_DIR_BASE} && \
    echo "INFO: SWIFT_URL is ${SWIFT_URL}" && \
    echo "INFO: Attempting to download Swift..." && \
    curl -L --verbose --fail --show-error -o ${SWIFT_INSTALL_DIR_BASE}/swift.tar.gz ${SWIFT_URL} && \
    echo "INFO: Swift download apparently successful. Extracting..." && \
    tar -xzf ${SWIFT_INSTALL_DIR_BASE}/swift.tar.gz -C ${SWIFT_INSTALL_DIR_BASE} && \
    echo "INFO: Swift extraction successful. Cleaning up..." && \
    rm ${SWIFT_INSTALL_DIR_BASE}/swift.tar.gz && \
    echo "INFO: Swift installation step completed."

# Add Swift to PATH
ENV PATH="${SWIFT_INSTALL_DIR_BASE}/${SWIFT_DIR_NAME}/usr/bin:${PATH}"
