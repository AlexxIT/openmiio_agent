@echo off

@SET GOOS=linux
@SET GOARCH=%1
@SET GOARM=7
@SET FILENAME=openmiio_agent

go build -ldflags "-s -w" -trimpath -o %FILENAME% && upx-3.95 --best --lzma %FILENAME%

if not "%2"=="" (
    rem Upload binary to gateway if pass gate IP-address as first param
    rem tcpsvd -E 0.0.0.0 21 ftpd -w &
    ftp -s:scripts\ftp.txt %2
)
