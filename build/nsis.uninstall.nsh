Section "Uninstall"
  # uninstall for all users
  setShellVarContext all

  # Delete (optionally) installed files
  {{range $}}Delete $INSTDIR\{{.}}
  {{end}}
  Delete $INSTDIR\uninstall.exe

  # Delete install directory
  rmDir $INSTDIR

  # Delete start menu launcher
  Delete "$SMPROGRAMS\${APPNAME}\${APPNAME}.lnk"
  Delete "$SMPROGRAMS\${APPNAME}\Attach.lnk"
  Delete "$SMPROGRAMS\${APPNAME}\Uninstall.lnk"
  rmDir "$SMPROGRAMS\${APPNAME}"

  # Firewall - remove rules if exists
  SimpleFC::AdvRemoveRule "Gpop incoming peers (TCP:60606)"
  SimpleFC::AdvRemoveRule "Gpop outgoing peers (TCP:60606)"
  SimpleFC::AdvRemoveRule "Gpop UDP discovery (UDP:60606)"

  # Remove IPC endpoint (https://github.com/popcateum/EIPs/issues/147)
  ${un.EnvVarUpdate} $0 "POPCATEUM_SOCKET" "R" "HKLM" "\\.\pipe\gpop.ipc"

  # Remove install directory from PATH
  Push "$INSTDIR"
  Call un.RemoveFromPath

  # Cleanup registry (deletes all sub keys)
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${GROUPNAME} ${APPNAME}"
SectionEnd
