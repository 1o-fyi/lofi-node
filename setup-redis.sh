#!/bin/bash
readonly __EXEC_DIR=$(dirname "$(realpath $0)") && cd $__EXEC_DIR

declare -a DEPS=( 
                    "pkg-config"
                    "gcc"
                    "make"
                    "autoconf"
                    "autogen"
                    "curl"
                    "wget"
                    "ca-certificates"
                    "openssl"
                    "tcl"
                    "git"
                    "libssl-dev"
                    "libjemalloc-dev"
                    "gnutls-bin"
                    "tclcurl"
                    "acl"
                    "libacl1-dev"
                    "libtool"
                    "dnsmasq-base"
                    "libgbtools-dev"
                    "gawk"
                )

apt install -y ${DEPS[@]} 
apt update -y 
apt upgrade -y 
apt autoremove -y

tarfile="redis-stable.tar.gz"
url="https://download.redis.io/redis-stable.tar.gz"
checksum_url="https://download.redis.io/redis-stable.tar.gz.SHA256SUM"

REDISPORT=6379
EXEC=/usr/local/bin/redis-server
CLIEXEC=/usr/local/bin/redis-cli
PIDFILE=/var/run/redis_${REDISPORT}.pid
CONF="/etc/redis/${REDISPORT}.conf"

cd /
#rm -rf $tarfile
#wget $url
#curl https://download.redis.io/redis-stable.tar.gz.SHA256SUM > checksum.new
#if [[ $(cat checksum.new) != $(cat redis-stable.tar.gz.SHA256SUM) ]]; then exit 1; fi

tar -xzf $tarfile
cp /redis-stable/redis.conf /etc/redis/${REDISPORT}.conf
mkdir -p /var/redis
mkdir -p /etc/redis
mkdir -p /var/redis/${REDISPORT}

cd /redis-stable
make distclean
make test
ln -s /redis-stable/src/redis-server /usr/local/bin/redis-server
ln -s /redis-stable/src/redis-cli /usr/local/bin/redis-cli
ln -s /redis-stable/utils/redis_init_script /etc/init.d/redis_${REDISPORT}

update-rc.d redis_${REDISPORT} defaults

echo -e "	You could try: \n\t/etc/init.d/redis_${REDISPORT} start\nOR\n\tredis-server --protected-mode no --daemonize yes"
