#!/bin/sh
# Probe NDMS user database / CLI on the router
export PATH=/opt/sbin:/opt/bin:$PATH
echo "=== ndmc show user"
ndmc -c "show user" 2>&1 | head -25
echo "=== ndmc help (user-related)"
ndmc -c "show" 2>&1 | head -30
echo "=== smb component"
ndmc -c "show smb" 2>&1 | head -20
echo "=== passwd entries"
cat /etc/passwd 2>/dev/null | head -20
echo "=== ndm dirs"
ls -la /opt/etc/ndm 2>/dev/null | head -10
find /tmp -maxdepth 2 -iname "*user*" 2>/dev/null | head -10
echo "=== ndm config files"
ls -la /tmp/ndm* 2>/dev/null | head -10
