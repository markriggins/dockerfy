FROM nginx:1.9
MAINTAINER Mark Riggins mark.riggins@gmail.com

RUN apt-get update -qq && apt-get install -y vim

# RUN wget https://github.com/markriggins/dockerfy/releases/download/v0.2.0/dockerfy-linux-amd64-v0.0.4.tar.gz
# RUN tar -C /usr/local/bin -xvzf dockerfy-linux-amd64-v0.2.0.tar.gz

COPY .dockerfy_linux_amd64.tar.gz /tmp/
RUN  cd /usr/local/bin && \
    tar -xzv --strip-components=1 -f /tmp/.d*.gz dockerfy_linux_amd64/dockerfy && \
    rm /tmp/.d*.gz &&  \
    echo "daemon off;" >> /etc/nginx/nginx.conf

COPY default.conf.tmpl /etc/nginx/conf.d/default.conf.tmpl

COPY overlays /tmp/overlays
COPY zombie-maker-debian-binary /usr/local/bin/zombie-maker
 
EXPOSE 80

ENV DEPLOYMENT_ENV=staging

#  NOTE: These options do not work well on virtualbox osx -- it pegs one CPU
#     "-log-poll", \
#     "-stdout", "/var/log/nginx/access.log", \
#     "-stderr", "/var/log/nginx/error.log", \

ENTRYPOINT [ "dockerfy", \
     "-template", "/etc/nginx/conf.d/default.conf.tmpl:/etc/nginx/conf.d/default.conf", \
     "-log-poll", \
     "-stdout", "/var/log/nginx/access.log", \
     "-stderr", "/var/log/nginx/error.log" ]


CMD [ "--", "nginx"]
