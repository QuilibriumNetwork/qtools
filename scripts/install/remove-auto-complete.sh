#!/bin/bash
# HELP: Removes tab autocomplete for the qtools command from bashrc.
log "Removing autocomplete for qtools command..."

# Remove the function-based completion block (multi-line removal)
sudo sed -i '/^_qtools_complete()/,/^complete -F _qtools_complete qtools$/d' $BASHRC_FILE

# Remove old simple completion (in case it exists from older installs)
pattern="^complete -W '.*' qtools$"
remove_lines_matching_pattern $BASHRC_FILE "$pattern"

# Remove the bash_completion source line
remove_lines_matching_pattern $BASHRC_FILE "^source /etc/profile.d/bash_completion.sh$"

source $BASHRC_FILE

log "Finished removing auto-complete."
