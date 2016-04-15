FROM nginx:1.9
MAINTAINER Mark Riggins mark.riggins@gmail.com

RUN apt-get update -qq && apt-get install -y vim

# Clients would install the latest from https://github.com/markriggins/dockerfy/releases,
# but we want to test what we just built locally
COPY .dockerfy-linux-amd64 /usr/local/bin/dockerfy

COPY default.conf.tmpl /etc/nginx/conf.d/default.conf.tmpl

COPY overlays /tmp/overlays
COPY zombie-maker-debian-binary /usr/local/bin/zombie-maker
 
EXPOSE 80

ENV DEPLOYMENT_ENV=staging

#  NOTE: These options do not work well on virtualbox osx -- it pegs one CPU
#     "--log-poll", \
#     "--stdout", "/var/log/nginx/access.log", \
#     "--stderr", "/var/log/nginx/error.log", \

ENTRYPOINT [ "dockerfy", \
     "--template", "/etc/nginx/conf.d/default.conf.tmpl:/etc/nginx/conf.d/default.conf", \
     "--log-poll", \
     "--stdout", "/var/log/nginx/access.log", \
     "--stderr", "/var/log/nginx/error.log" ]


CMD [ "--", "nginx"]
