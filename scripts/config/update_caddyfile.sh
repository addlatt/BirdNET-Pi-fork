#!/usr/bin/env bash
source /etc/birdnet/birdnet.conf
set -x
[ -d /etc/caddy ] || mkdir /etc/caddy
if [ -f /etc/caddy/Caddyfile ];then
  cp /etc/caddy/Caddyfile{,.original}
fi

# Web application root
WEB_ROOT="${HOME}/BirdNET-Pi/src/web"

if ! [ -z ${CADDY_PWD} ];then
HASHWORD=$(caddy hash-password --plaintext ${CADDY_PWD})
cat << EOF > /etc/caddy/Caddyfile
http:// ${BIRDNETPI_URL} {
  # Static assets
  handle /assets/* {
    root * ${WEB_ROOT}/public
    file_server
  }

  # Vendored tools (protected)
  handle /adminer/* {
    basicauth {
      birdnet ${HASHWORD}
    }
    root * ${WEB_ROOT}/vendor/adminer
    php_fastcgi unix//run/php/php-fpm.sock
  }
  handle /filemanager/* {
    basicauth {
      birdnet ${HASHWORD}
    }
    root * ${WEB_ROOT}/vendor/filemanager
    php_fastcgi unix//run/php/php-fpm.sock
  }

  # phpsysinfo (protected)
  handle /phpsysinfo/* {
    basicauth {
      birdnet ${HASHWORD}
    }
    root * ${HOME}/phpsysinfo
    php_fastcgi unix//run/php/php-fpm.sock
  }

  # Bird recordings (browse)
  handle /By_Date/* {
    root * ${EXTRACTED}
    file_server browse
  }
  handle /Charts/* {
    root * ${EXTRACTED}
    file_server browse
  }

  # Live spectrogram image
  handle /spectrogram.png {
    root * ${RECS_DIR}/StreamData
    file_server
  }

  # Protected stream
  basicauth /stream {
    birdnet ${HASHWORD}
  }
  reverse_proxy /stream localhost:8000

  # Protected terminal
  basicauth /terminal* {
    birdnet ${HASHWORD}
  }
  reverse_proxy /terminal* localhost:8888

  # Other reverse proxies
  reverse_proxy /log* localhost:8080
  reverse_proxy /stats* localhost:8501

  # PHP front controller (default handler)
  handle {
    root * ${WEB_ROOT}/public
    php_fastcgi unix//run/php/php-fpm.sock {
      try_files {path} /index.php
    }
    file_server
  }
}
EOF
else
cat << EOF > /etc/caddy/Caddyfile
http:// ${BIRDNETPI_URL} {
  # Static assets
  handle /assets/* {
    root * ${WEB_ROOT}/public
    file_server
  }

  # Vendored tools (no auth)
  handle /adminer/* {
    root * ${WEB_ROOT}/vendor/adminer
    php_fastcgi unix//run/php/php-fpm.sock
  }
  handle /filemanager/* {
    root * ${WEB_ROOT}/vendor/filemanager
    php_fastcgi unix//run/php/php-fpm.sock
  }

  # phpsysinfo (no auth)
  handle /phpsysinfo/* {
    root * ${HOME}/phpsysinfo
    php_fastcgi unix//run/php/php-fpm.sock
  }

  # Bird recordings (browse)
  handle /By_Date/* {
    root * ${EXTRACTED}
    file_server browse
  }
  handle /Charts/* {
    root * ${EXTRACTED}
    file_server browse
  }

  # Live spectrogram image
  handle /spectrogram.png {
    root * ${RECS_DIR}/StreamData
    file_server
  }

  # Reverse proxies
  reverse_proxy /stream localhost:8000
  reverse_proxy /log* localhost:8080
  reverse_proxy /stats* localhost:8501
  reverse_proxy /terminal* localhost:8888

  # PHP front controller (default handler)
  handle {
    root * ${WEB_ROOT}/public
    php_fastcgi unix//run/php/php-fpm.sock {
      try_files {path} /index.php
    }
    file_server
  }
}
EOF
fi

sudo caddy fmt --overwrite /etc/caddy/Caddyfile
sudo systemctl reload caddy
