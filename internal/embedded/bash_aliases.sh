alias nano="vi"
alias please='sudo $(fc -ln -1)'
alias refresh='source ~/.bashrc; echo "Reloaded .bashrc."'
alias reload='source ~/.bashrc; echo "Reloaded .bashrc."'

if type batcat &>/dev/null; then
  alias cat="batcat --paging=never"
  alias catp="batcat --paging=always"
elif type bat &>/dev/null; then
  alias cat="bat --paging=never"
  alias catp="bat --paging=always"
fi

if type eza &>/dev/null; then
  alias ls="eza --icons"
  alias ll="eza -lg --icons"
  alias la="eza -lag --icons"
  alias lt="eza -lTg --icons"
  alias lt1="eza -lTg --level=1 --icons"
  alias lt2="eza -lTg --level=2 --icons"
  alias lt3="eza -lTg --level=3 --icons"
fi

if type lazygit &>/dev/null; then
  alias lg="lazygit"
fi
