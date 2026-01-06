#!/usr/bin/env bash
source /etc/birdnet/birdnet.conf
set -x
[ -d /etc/caddy ] || mkdir /etc/caddy
if [ -f /etc/caddy/Caddyfile ];then
  cp /etc/caddy/Caddyfile{,.original}
fi

# Derive user home from RECS_DIR (e.g., /home/addlatt/BirdSongs -> /home/addlatt)
# This avoids issues with ${HOME} expanding to /root when run with sudo
USER_HOME=$(dirname "${RECS_DIR}")

# Web application root
WEB_ROOT="${USER_HOME}/BirdNET-Pi/src/web"

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
    root * ${USER_HOME}/phpsysinfo
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
    root * ${EXTRACTED}
    file_server
  }

  # Protected stream
  handle /stream {
    basicauth {
      birdnet ${HASHWORD}
    }
    reverse_proxy localhost:8000
  }

  # Protected terminal
  handle /terminal* {
    basicauth {
      birdnet ${HASHWORD}
    }
    reverse_proxy localhost:8888
  }

  # Other reverse proxies
  handle /log* {
    reverse_proxy localhost:8080
  }
  handle /stats* {
    reverse_proxy localhost:8501
  }

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
    root * ${USER_HOME}/phpsysinfo
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
    root * ${EXTRACTED}
    file_server
  }

  # Reverse proxies
  handle /stream {
    reverse_proxy localhost:8000
  }
  handle /log* {
    reverse_proxy localhost:8080
  }
  handle /stats* {
    reverse_proxy localhost:8501
  }
  handle /terminal* {
    reverse_proxy localhost:8888
  }

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
