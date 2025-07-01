# Copyright 2024-2025 NetCracker Technology Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.24.3-alpine3.21 AS builder
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

COPY ./ ./

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build -o duplicator ./main.go

# Copy to vanilla alpine container
FROM alpine:3.22

ENV USER_UID="1001" \
    USER_NAME="duplicator" \
    GROUP_NAME="duplicator"

COPY --from=builder /workspace/duplicator /

RUN \
    # Add user
    addgroup ${GROUP_NAME} \
    && adduser -D -G ${GROUP_NAME} -u ${USER_UID} ${USER_NAME} \
    # Grant execute permissions
    && chmod a+x /duplicator

USER ${USER_UID}

CMD ["/duplicator"]
