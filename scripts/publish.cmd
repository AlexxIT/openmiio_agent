@echo off

set SF_USER=%1
set SF_PASS=%2

call scripts/build.cmd arm
python.exe scripts/sourceforge.py /aqcn02/openmiio_agent/openmiio_agent

call scripts/build.cmd mipsle
python.exe scripts/sourceforge.py /mgl03/openmiio_agent/openmiio_agent