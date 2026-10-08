SKIPUNZIP=0

ui_print "****************************************"
ui_print " Droidspaces All-in-One Engine & WebUI "
ui_print " 100% Standalone Headless Container Host"
ui_print "****************************************"

DROIDSPACES_DIR="/data/local/Droidspaces"
mkdir -p "$DROIDSPACES_DIR/bin"
mkdir -p "$DROIDSPACES_DIR/Logs"
mkdir -p "$DROIDSPACES_DIR/Containers"
mkdir -p "$DROIDSPACES_DIR/rootfs"

# Install native core C binaries directly to /data/local/Droidspaces/bin/
if [ -f "$MODPATH/bin/droidspaces" ]; then
    cp -f "$MODPATH/bin/droidspaces" "$DROIDSPACES_DIR/bin/droidspaces"
    chmod 755 "$DROIDSPACES_DIR/bin/droidspaces"
fi

if [ -f "$MODPATH/bin/busybox" ]; then
    cp -f "$MODPATH/bin/busybox" "$DROIDSPACES_DIR/bin/busybox"
    chmod 755 "$DROIDSPACES_DIR/bin/busybox"
fi

# Install helper scripts into /data/local/Droidspaces/bin/
for script in post_extract_fixes.sh sparsemgr.sh export_container.sh; do
    if [ -f "$MODPATH/$script" ]; then
        cp -f "$MODPATH/$script" "$DROIDSPACES_DIR/bin/$script"
        chmod 755 "$DROIDSPACES_DIR/bin/$script"
    fi
done

# Enable daemon mode flag so core daemon starts on boot
echo "1" > "$DROIDSPACES_DIR/.daemon_mode"

# Permissions
set_perm_recursive "$MODPATH" 0 0 0755 0644
set_perm "$MODPATH/service.sh" 0 0 0755
set_perm "$MODPATH/post-fs-data.sh" 0 0 0755
[ -f "$MODPATH/bin/droidspaces" ] && set_perm "$MODPATH/bin/droidspaces" 0 0 0755
[ -f "$MODPATH/bin/busybox" ] && set_perm "$MODPATH/bin/busybox" 0 0 0755
[ -f "$MODPATH/bin/dsweb_arm64" ] && set_perm "$MODPATH/bin/dsweb_arm64" 0 0 0755
[ -f "$MODPATH/post_extract_fixes.sh" ] && set_perm "$MODPATH/post_extract_fixes.sh" 0 0 0755
[ -f "$MODPATH/sparsemgr.sh" ] && set_perm "$MODPATH/sparsemgr.sh" 0 0 0755
[ -f "$MODPATH/export_container.sh" ] && set_perm "$MODPATH/export_container.sh" 0 0 0755

ui_print "- Core C Engine installed."
ui_print "- Svelte 5 WebUI Dashboard installed."
ui_print "- Access after boot at http://<phone-ip>:84"
