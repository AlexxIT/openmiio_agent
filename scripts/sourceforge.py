"""
Upload binary to sourceforge.net with username and password from os environments.

https://sourceforge.net/p/forge/documentation/Release%20Files%20for%20Download/#scp
"""
import hashlib
import io
import os
import pathlib
import sys

from paramiko import SSHClient, client
from scp import SCPClient

ssh = SSHClient()
ssh.set_missing_host_key_policy(client.AutoAddPolicy())
ssh.connect(
    "frs.sourceforge.net",
    username=os.environ["SF_USER"],
    password=os.environ["SF_PASS"],
)

raw = open(sys.argv[1], "rb").read()
f = io.BytesIO(raw)
f.seek(0)

scp = SCPClient(ssh.get_transport())
scp.putfo(f, sys.argv[2])
scp.close()
