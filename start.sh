#!/bin/sh
cd /app
go run server.go --imapserver $IMAPSERVER --imapport $IMAPPORT --imapdomain $IMAPDOMAIN --clientdomain $CLIENTDOMAIN --clientid $CLIENTID --clientsecret $CLIENTSECRET
