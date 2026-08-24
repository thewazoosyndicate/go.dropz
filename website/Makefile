IMAGE := registry.desku.be/epheo/dropz-web
TAG   := $(shell date +%Y%m%d)-$(shell git rev-parse --short HEAD)
GITOPS := ../../../defms/defms-k8s-gitops/applications/dropz-web/kustomization.yaml

.PHONY: release build run push deploy

release:
	./release.sh

build:
	podman build -t $(IMAGE):$(TAG) -t $(IMAGE):latest .

run: build
	podman run --rm -p 8080:8080 --read-only $(IMAGE):$(TAG)

push: build
	podman push $(IMAGE):$(TAG)
	podman push $(IMAGE):latest

# ArgoCD only rolls the pod when the manifest changes, so pin the tag.
deploy: push
	sed -i 's/^\(    newTag: \).*/\1$(TAG)/' $(GITOPS)
	@echo "bumped $(GITOPS) to $(TAG); commit and push the gitops repo"
