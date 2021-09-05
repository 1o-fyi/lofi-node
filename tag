#!/bin/bash
if [[ $EUID -eq 0 ]]; then echo -e "\nThis script shouldn't run as root" && exit 1; fi

readonly __EXEC_DIR=$(dirname "$(realpath $0)") && cd $__EXEC_DIR
readonly __VERSION="$__EXEC_DIR/.tag";
readonly __MVAR="$__VERSION/MVAR";
readonly __MNOR="$__VERSION/MNOR";
readonly __RLSE="$__VERSION/RLSE";
readonly __CMD=${1:-"version"}
readonly __FLAG_MIN=0
readonly __FLAG_MAX=256
readonly __DIGEST="blake2s256"

function fmt-exec
{ 
	cat tag | grep "function *" | tail -n +8 | sed 's/function /tag /g' 
}

function fmt-src
{
	cat tag | tr -s '\t '
}

function fmt
{ 
	printf "%-2s\n" "$( fmt-exec | sed 's/tag /	/g' | columns -s -c 4 -w 1)" 
}

function version
{	# the default command - shows a semantic version tag
	if [ -d "$__VERSION" ]; then
	echo "v$(cat $__MVAR).$(cat $__MNOR).$(cat $__RLSE)"
	fi
}

function v
{	# shorthand for version, also the default command
	version
}

function v-
{ 	# Inverts the version
	echo "v$(version | rev | sed 's/v//g')"
}

function y
{ 	# Prints each flags value on a new line, in order
 	v | tr -d "v" | tr "." "\n" 
}


function y-
{	# Prints each flags value on a new line, in reverse
 	v- | tr -d "v" | tr "." "\n" 
}

function gen
{ 	# generates all possible flags
 	printf "v%s\n" {0..256}"."{0..256}"."{0..256}
}

function update-info
{ 
	fmt > USAGE 
}

function token
{
 	cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1 | openssl dgst -$__DIGEST | base64 -w 0 | jq -R '{ '"$__DIGEST"': '.' }'
}

function new-repo
{
	git init -b main
	git add . && git commit -m "auto-$( version )"
}

function clear
{	# removes the current version tag
	rm -rf $__VERSION
}

function __init
{	# initilizes the tag format
	if ! [ -d "$(pwd)/.tag" ]; then
	if ! [ -d ".git" ]; then new-repo; fi
	mkdir -p $__VERSION
	echo 0 > $__MVAR
	echo 0 > $__MNOR
	echo 0 > $__RLSE; fi 
}

function switch
{
	git switch -C "$(v)"
}

function inc
{	
	_vf=${1:-"$__RLSE"} &&
	i=$(cat $_vf) &&
	i=$(($((i))+1)) &&
	if [ $i -gt $__FLAG_MAX ]; then 
	i=$__FLAG_MIN
	fi
	echo $i > $_vf &&
	./tag "switch"
}

function dec
{	
	_vf=${1:-"$__RLSE"} && 
	i=$(cat $_vf) &&
	i=$(($((i))-1)) && 
	if [ $i -lt $__FLAG_MIN ]; then 
	i=$__FLAG_MAX
	fi
	echo $i > $_vf &&
	./tag "switch"
}


function incr 
{	
	inc $__RLSE
}

function decr 
{
	dec $__RLSE
}

function incm
{
 	inc $__MNOR
}

function decm
{
 	dec $__MNOR
}

function incM
{
 	inc $__MVAR
} 

function decM
{
 	dec $__MVAR
}

function curb
{	
	git branch --list | grep "* " | tr -d " *"
}

function is_in_remote
{
	local branch=${1}
	local existed_in_remote=$(git ls-remote --heads origin ${branch})
	if [[ -z ${existed_in_remote} ]]; then
	echo 0
	else
	echo 1
	fi
}

function is_in_local 
{
	local branch=${1}
	local existed_in_local=$(git branch --list ${branch})
	if [[ -z ${existed_in_local} ]]; then
	echo 0
	else
	echo 1
	fi
}

function sign
{
	git commit --amend -s
}

function email
{
	sendemail -t tag@lists.sr.ht
}

function apt-deps
{
 	apt install jq autogen git
}

__init && 
$__CMD
