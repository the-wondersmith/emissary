IMAGE_REPO ?= ghcr.io/emissary-ingress/emissary
CHART_DIR ?= $(OSS_HOME)/charts

CRDS_CHART = $(CHART_DIR)/emissary-crds-chart-$(VERSION).tgz

all: chart

version-check:
	@if [ -z "$(VERSION)" ]; then \
		echo "VERSION must be set (e.g. VERSION=1.0.0-alpha.3)" >&2 ;\
		exit 1; \
	fi
.PHONY: version-check

helm-registry-check:
	@if [ -z "$(HELM_REGISTRY)" ]; then \
		echo "HELM_REGISTRY must be set (e.g. HELM_REGISTRY=oci://docker.io/dwflynn)" >&2 ;\
		exit 1; \
	fi
.PHONY: helm-registry-check

$(CRDS_CHART): version-check $(CHART_DIR)/emissary-crds
	rm -rf $(CRDS_CHART)
	cp -pr $(CHART_DIR)/emissary-crds $(CHART_DIR)/emissary-crds-chart-$(VERSION)
	sed -e "s/%VERSION%/$(VERSION)/" \
		< $(CHART_DIR)/emissary-crds-chart-$(VERSION)/Chart.yaml > $(CHART_DIR)/emissary-crds-chart-$(VERSION)/Chart-fixed.yaml
	mv $(CHART_DIR)/emissary-crds-chart-$(VERSION)/Chart-fixed.yaml $(CHART_DIR)/emissary-crds-chart-$(VERSION)/Chart.yaml
	helm package ./$(CHART_DIR)/emissary-crds-chart-$(VERSION)

push-chart: version-check helm-registry-check $(CRDS_CHART)
	if [ -n "$(HELM_REGISTRY)" ]; then \
		helm push $(CRDS_CHART) $(HELM_REGISTRY); \
	else \
		echo "HELM_REGISTRY not set, not pushing"; \
	fi

# This is just an alias
chart: $(CRDS_CHART)

clean:
	rm -rf $(CHART_DIR)/emissary-crds-chart-*

