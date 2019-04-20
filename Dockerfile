FROM ccr.ccs.tencentyun.com/dhub.wallstcn.com/alpine:3.5

ENV CONFIGOR_ENV ivktest
RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
ADD server /
ADD conf/ /conf
#ADD public /public
ENTRYPOINT [ "/server" ]
