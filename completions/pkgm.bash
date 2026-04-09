# completions/pkgm.bash
# bash completion for pkgm

_pkgm() {
    local cur prev words cword
    _init_completion || return

    if [[ $cword -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "init apply status reset -h --help -V --version" -- "$cur") )
        return
    fi

    case "${words[1]}" in
        init)
            COMPREPLY=( $(compgen -W "-h --help" -- "$cur") )
            ;;
        apply)
            COMPREPLY=( $(compgen -W "-n --dry-run -h --help" -- "$cur") )
            ;;
        status)
            COMPREPLY=( $(compgen -W "-h --help" -- "$cur") )
            ;;
        reset)
            COMPREPLY=( $(compgen -W "--all -h --help" -- "$cur") )
            ;;
        version)
            COMPREPLY=()
            ;;
    esac
}

complete -F _pkgm pkgm
