HISTSIZE=25000
HISTFILESIZE=200000
HISTCONTROL=ignoreboth:erasedups
shopt -s histappend

__git_branch() {
  local branch
  branch="$(git symbolic-ref --short HEAD 2>/dev/null)"
  [ -n "$branch" ] && printf ' \033[33m(%s)\033[0m' "$branch"
}

PS1='\n\[\033[38;5;240m\]╭\[\033[0m\] \[\033[38;5;208m\]\w\[\033[0m\]$(__git_branch)\n\[\033[38;5;240m\]╰\[\033[0m\] \[\033[32m\]❯\[\033[0m\] '

export TERM=${TERM:-xterm-256color}
export LANG=${LANG:-en_US.UTF-8}
export LC_ALL=${LC_ALL:-$LANG}

alias grep="grep --color=auto"
alias fgrep="fgrep --color=auto"
alias egrep="egrep --color=auto"

[ -f "$HOME/.bash_aliases" ] && source "$HOME/.bash_aliases"
