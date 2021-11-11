CC := GO111MODULE=on CGO_ENABLED=0 go
CFLAGS := build -o
SHELL := /bin/bash
NAME := lofi-node
USER := 1o-fyi
REMOTE := github.com
MODULE := $(PWD)/$(REMOTE)/$(USER)/$(NAME)
RW := git@$(MODULE)
DAEMON_CONFIG := $(NAME).service
DAEMON_ENV := /etc/conf.d/$(NAME)
DAEMON_PATH := /var/local/$(NAME)
DAEMON_CONFIG_PATH := /etc/systemd/system/$(DAEMON_CONFIG)

VERSION := $(shell ./tag)

build :: copy-local

init :: build
				$(CC) mod init $(MODULE)

format :: 
				$(CC)fmt -w -s *.go

mod-install :: format
				$(CC) install ./... 

tidy :: mod-install
				$(CC) mod tidy -compat=1.17
				
test ::	 tidy
				$(CC) test -v ./...

compile :: test
				$(CC) $(CFLAGS) $(MODULE) && chmod 755 $(MODULE)

link-local :: compile
				$(shell ldd $(MODULE))

headers :: link-local
				$(shell readelf -h $(MODULE) > $(MODULE).headers)

copy-local :: headers
				cp $(MODULE) .

set-cap ::
				setcap $(shell echo Y2FwX3N5c19yYXdpbyxjYXBfZGFjX3JlYWRfc2VhcmNoLGNhcF9ta25vZCxjYXBfc3lzX25pY2UsY2FwX2lwY19sb2NrLGNhcF9kYWNfb3ZlcnJpZGUsY2FwX2F1ZGl0X3JlYWQsY2FwX3NldGZjYXAsY2FwX25ldF9hZG1pbixjYXBfd2FrZV9hbGFybSxjYXBfYXVkaXRfd3JpdGUsY2FwX25ldF9iaW5kX3NlcnZpY2U9K2VwCg== | base64 -d) $(NAME)

cap :: copy-local 
				$(MAKE) -s set-cap

init-service :: cap
				mkdir -p $(DAEMON_PATH) $(DAEMON_ENV)
				cp $(DAEMON_CONFIG) $(DAEMON_CONFIG_PATH)
				cp $(MODULE) $(DAEMON_PATH)/start
			
status ::
				systemctl status $(NAME)

start :: 
				systemctl start $(NAME)

enable :: start
				systemctl enable $(NAME)

disable :: 
				systemctl disable $(NAME)

stop :: disable
				systemctl stop $(NAME)

purge :: stop
				rm -rf $(MODULE) $(DAEMON_CONFIG_PATH) $(DAEMON_PATH)

replace ::
				mv $(NAME) $(DAEMON_PATH)/start

daemon-reload ::
				systemctl daemon-reload

reload :: replace daemon-reload enable status

logs ::
				journalctl --flush && journalctl -n 5

service :: init-service start
				systemctl daemon-reload

send ::
				cd .. && tar cf $(NAME).$(VERSION).tar.xz $(NAME)/ && wormhole send $(NAME).$(VERSION).tar.xz   

send ::
				cd .. && tar cf $(NAME).$(VERSION).tar.xz $(NAME)/ && wormhole send $(NAME).$(VERSION).tar.xz   

get-tag ::
				$(shell curl https://git.sr.ht/~johns/tag/blob/main/tag > tag && chmod 755 tag)

get-go ::
				$(shell curl https://git.sr.ht/~johns/install-go/blob/main/install-go > install-go && chmod 755 install-go)

get-license ::
				$(shell curl https://www.gnu.org/licenses/agpl-3.0.txt > LICENSE)

install-scripts :: get-tag get-go
