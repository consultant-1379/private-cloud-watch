#!/usr/bin/env bash

wget https://files.pythonhosted.org/packages/b6/ac/7015eb97dc749283ffdec1c3a88ddb8ae03b8fad0f0e611408f196358da3/pip-9.0.1-py2.py3-none-any.whl -P ./wheels
pip download -r requirements.txt -d ./wheels
