autoload -Uz compinit && compinit

setopt SHARE_HISTORY
setopt HIST_SAVE_NO_DUPS
setopt HIST_IGNORE_ALL_DUPS
setopt HIST_IGNORE_SPACE
setopt HIST_VERIFY
HISTSIZE=25000
SAVEHIST=200000

autoload -Uz vcs_info
precmd() { vcs_info }
zstyle ':vcs_info:git:*' formats ' %F{yellow}(%b)%f'
setopt PROMPT_SUBST
PROMPT='%F{208}%~%f${vcs_info_msg_0_} %F{green}➜%f '

[ -f /usr/share/zsh-autosuggestions/zsh-autosuggestions.zsh ] && source /usr/share/zsh-autosuggestions/zsh-autosuggestions.zsh
[ -f /usr/share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh ] && source /usr/share/zsh-syntax-highlighting/zsh-syntax-highlighting.zsh

export LANG=${LANG:-en_US.UTF-8}
export LC_ALL=${LC_ALL:-$LANG}

alias grep="grep --color=auto"
alias fgrep="fgrep --color=auto"
alias egrep="egrep --color=auto"

[ -f "$HOME/.zshrc_aliases" ] && source "$HOME/.zshrc_aliases"
