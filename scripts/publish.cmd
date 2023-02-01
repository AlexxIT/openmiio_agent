@echo off

set SF_USER=%1
set SF_PASS=%2

python.exe scripts/sourceforge.py openmiio_agent_mips /home/frs/project/mgl03/openmiio_agent/openmiio_agent-1.0.1
python.exe scripts/sourceforge.py openmiio_agent_arm /home/frs/project/aqcn02/openmiio_agent/openmiio_agent-1.0.1

certutil -hashfile openmiio_agent_mips md5
certutil -hashfile openmiio_agent_arm md5
