@echo off

@SET GOOS=linux
@SET GOARCH=mipsle
@SET FILENAME=openmiio_agent_mips
go build -ldflags "-s -w" -trimpath -o %FILENAME% && upx-3.95 --best --lzma %FILENAME%

@SET GOARCH=arm
@SET GOARM=7
@SET FILENAME=openmiio_agent_arm
go build -ldflags "-s -w" -trimpath -o %FILENAME% && upx-3.95 --best --lzma %FILENAME%
