#!/usr/bin/env python

# Copyright (c) Microsoft Corporation.
# Licensed under the Apache License 2.0.

from codecs import open
from setuptools import setup, find_packages
try:
    from azure_bdist_wheel import cmdclass
except ImportError:
    from distutils import log as logger
    logger.warn("Wheel is not available, disabling bdist_wheel hook")

VERSION = '1.0.11'

# The full list of classifiers is available at
# https://pypi.python.org/pypi?%3Aaction=list_classifiers
CLASSIFIERS = [
    'Development Status :: 4 - Beta',
    'Intended Audience :: Developers',
    'Intended Audience :: System Administrators',
    'Programming Language :: Python',
    'Programming Language :: Python :: 3',
    'Programming Language :: Python :: 3.6',
    'Programming Language :: Python :: 3.7',
    'Programming Language :: Python :: 3.8',
    'License :: OSI Approved :: Apache Software License',
]

DEPENDENCIES = [
    'azure-cli-core'
]

with open('README.rst', 'r', encoding='utf-8') as f:
    README = f.read()
with open('HISTORY.rst', 'r', encoding='utf-8') as f:
    HISTORY = f.read()

setup(
    name='aro',
    version=VERSION,
    description='Microsoft Azure Command-Line Tools ARO Extension',
    author='Red Hat, Inc.',
    author_email='support@redhat.com',
    url='https://github.com/Azure/ARO-RP',
    long_description=README + '\n\n' + HISTORY,
    license='Apache',
    classifiers=CLASSIFIERS,
    packages=find_packages(),
    install_requires=DEPENDENCIES,
    package_data={'azext_aro': ['azext_metadata.json']},
)
