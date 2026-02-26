#!/bin/bash
# HELP: Removes tab autocomplete for the qtools command from bashrc.
log "Removing autocomplete for qtools command..."

if grep -q '_qtools_complete' $BASHRC_FILE 2>/dev/null; then
  # Use awk to skip the entire block from _qtools_complete() { to complete -F _qtools_complete qtools
  sudo awk '
    /_qtools_complete\(\)/ { skip=1 }
    skip && /complete -F _qtools_complete qtools/ { skip=0; next }
    !skip
  ' $BASHRC_FILE > /tmp/bashrc_cleaned && sudo mv /tmp/bashrc_cleaned $BASHRC_FILE
  log "Removed _qtools_complete function block."
else
  log "_qtools_complete not found in $BASHRC_FILE, skipping."
fi

# Remove old simple completion (in case it exists from older installs)
if grep -qE "^complete -W '.*' qtools$" $BASHRC_FILE 2>/dev/null; then
  remove_lines_matching_pattern $BASHRC_FILE "^complete -W '.*' qtools$"
fi

# Remove the bash_completion source line
if grep -q "source /etc/profile.d/bash_completion.sh" $BASHRC_FILE 2>/dev/null; then
  sudo sed -i '\|source /etc/profile.d/bash_completion.sh|d' $BASHRC_FILE
fi

source $BASHRC_FILE

log "Finished removing auto-complete."
