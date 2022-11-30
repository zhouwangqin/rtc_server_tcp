#!/bin/bash

APP_DIR=$(cd `dirname $0`/../; pwd)
echo "$APP_DIR"
cd $APP_DIR

BIZ=biz
ISLB=islb
SFU=sfu

BUILD_PATH1=$APP_DIR/bin/$BIZ
BUILD_PATH2=$APP_DIR/bin/$ISLB
BUILD_PATH3=$APP_DIR/bin/$SFU

BIZ_LOG=$APP_DIR/logs/$BIZ.log
ISLB_LOG=$APP_DIR/logs/$ISLB.log
SFU_LOG=$APP_DIR/logs/$SFU.log

echo "------------------delete $BIZ------------------"
echo "rm $BUILD_PATH1"
rm $BUILD_PATH1

echo "------------------delete $ISLB------------------"
echo "rm $BUILD_PATH2"
rm $BUILD_PATH2

echo "------------------delete $SFU------------------"
echo "rm $BUILD_PATH3"
rm $BUILD_PATH3

echo "------------------delete $BIZ LOG------------------"
echo "rm $BIZ_LOG"
rm $BIZ_LOG

echo "------------------delete $ISLB LOG------------------"
echo "rm $ISLB_LOG"
rm $ISLB_LOG

echo "------------------delete $SFU LOG------------------"
echo "rm $SFU_LOG"
rm $SFU_LOG
