#!/usr/bin/env python

import sys

import argparse
import os
import yaml


def indent(s, level):
    istr = " " * level

    return "\n".join([ f"{istr}{line}" for line in s.splitlines() ])


def main():
    parser = argparse.ArgumentParser(
        description="Split CRD YAML files into individual Helm chart templates"
    )
    parser.add_argument(
        "--output", "-o",
        required=True,
        help="Output directory for the split CRD files"
    )
    parser.add_argument(
        "crd_files",
        nargs="+",
        help="One or more paths to CRD YAML files to process"
    )

    args = parser.parse_args()

    # Create output directory if it doesn't exist
    os.makedirs(args.output, exist_ok=True)

    for path in args.crd_files:
        crd = yaml.safe_load(open(path))

        # Make sure it's really a CRD.
        if crd['kind'] != 'CustomResourceDefinition':
            continue

        # Grab the name of the CRD.
        name = crd['metadata']['name']
        shortname = name.replace(".getambassador.io", "")

        versions = {}

        for version in crd['spec']['versions']:
            versions[version['name']] = version

        output_path = os.path.join(args.output, f"{shortname}.yaml")
        output = open(output_path, 'w')

        print(f"{name}: {output_path}")

        metadata = crd['metadata']

        if "labels" not in metadata:
            metadata["labels"] = {}

        if "helm.sh/chart" not in metadata["labels"]:
            metadata["labels"]["helm.sh/chart"] = "XXXLABEL-HELM-CHARTXXX"

        if "emissary-ingress.dev/control-plane-ns" not in metadata["labels"]:
            metadata["labels"]["emissary-ingress.dev/control-plane-ns"] = "XXXLABEL-CONTROL-PLANE-NSXXX"

        info = {
            'apiVersion': crd['apiVersion'],
            'kind': crd['kind'],
            'metadata': metadata,
            }

        s = yaml.safe_dump(info)

        s = s.replace("XXXLABEL-HELM-CHARTXXX", '{{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}')
        s = s.replace("XXXLABEL-CONTROL-PLANE-NSXXX", '{{ .Release.Namespace }}')

        output.write(s)
        output.write(f"spec:\n")
        output.write('  {{- include "partials.conversion" . }}\n')

        n = crd["spec"]["names"]
        categories = n.get('categories')

        if categories:
            n['categories'] = categories
        else:
            n['categories'] = [ "ambassador-crds" ]

        info = {
            "group": crd['spec']['group'],
            "names": crd['spec']['names'],
            "scope": crd['spec']['scope'],
            "preserveUnknownFields": crd['spec'].get('preserveUnknownFields', False),
        }

        output.write(indent(yaml.safe_dump(info), 2))
        output.write("\n")

        output.write("  versions:\n")

        if "v1" in versions or "v2" in versions:
            output.write("{{- if .Values.enableLegacyVersions }}\n")

        if "v1" in versions:
            output.write("{{- if .Values.enableV1 }}\n")
            output.write(indent(yaml.safe_dump([ versions["v1"] ]), 2))
            output.write("\n")
            output.write("{{- end }}\n")

        if "v2" in versions:
            output.write(indent(yaml.safe_dump([ versions["v2"] ]), 2))
            output.write("\n")
            output.write("{{- end }}\n")

        if "v3alpha1" in versions:
            v = versions["v3alpha1"]
            v["storage"] = "XXXSTORAGEXXX"

            s = indent(yaml.safe_dump([ v ]), 2)
            s = s.replace("XXXSTORAGEXXX", '{{ include "partials.v3alpha1storage" . }}')

            output.write(s)
            output.write("\n")

        output.close()


if __name__ == "__main__":
    main()

