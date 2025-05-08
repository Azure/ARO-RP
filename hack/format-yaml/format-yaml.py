#!/usr/bin/env python3

# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

import os
import sys
import ruamel.yaml


def main():
    yaml = ruamel.yaml.YAML()

    for root, _, files in os.walk(sys.argv[1]):
        for name in files:
            if not name.endswith('.yml'):
                continue

            with open(os.path.join(root, name)) as f:
                y = yaml.load(f)

            with open(os.path.join(root, name), 'w') as f:
                yaml.dump(y, f)


if __name__ == '__main__':
    main()
