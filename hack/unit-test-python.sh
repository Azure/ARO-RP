#!/bin/bash

# This script executes Python unit tests. It first tries to activate the virtual environment, 
# but if it does not exist it will be created and then activated.
# When executing "pytest" command, it ignores integration test folder as 
# this script is intended to just execute unit tests. 

. pyenv/bin/activate || python3 -m venv pyenv && . pyenv/bin/activate
pip install pytest
cd python
pytest --ignore=az/aro/azext_aro/tests/latest/integration
