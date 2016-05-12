#
#   Build an nginx image with dockerfy and zombie-maker strictly for test purposes
#   
#   NOTE:  this is NOT an example of how YOU should use dockerfy in a Dockerfile
#

# Start with a locally built nginx-with-dockerfy

FROM nginx-with-dockerfy
MAINTAINER Mark Riggins mark.riggins@gmail.com

# RUN apt-get update -qq && apt-get install -y vim

COPY default.conf.tmpl /etc/nginx/conf.d/default.conf.tmpl

COPY overlays /tmp/overlays
COPY .zombie-maker-debian-binary /usr/local/bin/zombie-maker

# normally /secrets would be a mounted volume
COPY secrets.json /secrets/secrets.json
COPY secrets.2.json /secrets/secrets.2.json
 
EXPOSE 80

ENV DEPLOYMENT_ENV=staging

ENTRYPOINT [ "dockerfy", \
     "--template", "/etc/nginx/conf.d/default.conf.tmpl:/etc/nginx/conf.d/default.conf", \
     "--log-poll", \
     "--stdout", "/var/log/nginx/access.log", \
     "--stderr", "/var/log/nginx/error.log" ]


CMD [ "--", "nginx"]
