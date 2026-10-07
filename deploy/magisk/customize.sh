SKIPUNZIP=0

ui_print "- Installing Droidspaces WebUI Module..."
ui_print "- Configured listening port: 84"
mkdir -p "$MODPATH/bin"
set_perm_recursive "$MODPATH" 0 0 0755 0644
set_perm "$MODPATH/service.sh" 0 0 0755
set_perm "$MODPATH/bin/dsweb_arm64" 0 0 0755
ui_print "- WebUI will start on http://<phone-ip>:84 after boot."
