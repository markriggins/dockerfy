#
#  Build the ready-to-use nginx-with-dockerfy image to push to dockerhub
#
#  NOTE: this is not an example of how to use the nginx-with-dockerfy image -- see examples/* for examples
#
FROM nginx:1.9

COPY dist/linux/amd64/dockerfy /usr/local/bin/dockerfy

ENTRYPOINT [ 'dockerfy' ]

CMD [ "--", nginx",  "-g",  "daemon off;" ]

