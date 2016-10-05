FROM socialcode/nginx-with-dockerfy

COPY default.conf.tmpl /etc/nginx/conf.d/default.conf.tmpl

ENTRYPOINT [ "dockerfy", \
     "--template", "/etc/nginx/conf.d/default.conf.tmpl:/etc/nginx/conf.d/default.conf", \
     "nginx",  "-g",  "daemon off;" ]

