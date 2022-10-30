"""
Upload binary to sourceforge.net with username and password from os environments. And prints md5 of binary.

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
    'frs.sourceforge.net',
    username=os.environ['SF_USER'],
    password=os.environ['SF_PASS']
)

scp = SCPClient(ssh.get_transport())

raw = open('openmiio_agent', 'rb').read()
hex_ = hashlib.md5(raw).hexdigest()
print(sys.argv[1], hex_)

f = io.BytesIO(raw)
f.seek(0)
scp.putfo(f, '/home/frs/project' + sys.argv[1])
scp.close()
