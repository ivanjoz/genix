#!/bin/bash
if [[ "$OSTYPE" == "linux-gnu"* ]]; then # comando para LINUX

  if [ -x /usr/bin/qdbus-qt5 ]; then
      QDBUS_CMD="/usr/bin/qdbus-qt5"
  elif [ -x /usr/bin/qdbus-qt6 ]; then
      QDBUS_CMD="/usr/bin/qdbus-qt6"
  else
      QDBUS_CMD="qdbus"
  fi

  konsole & sleep 1

  $QDBUS_CMD org.kde.konsole-$! /konsole/MainWindow_1 org.kde.KMainWindow.activateAction split-view-left-right

  $QDBUS_CMD org.kde.konsole-$! /Windows/1 org.kde.konsole.Window.setCurrentSession 1
  $QDBUS_CMD org.kde.konsole-$! /Sessions/1 setTitle 1 'FRONTEND'
  $QDBUS_CMD org.kde.konsole-$! /Sessions/1 runCommand 'cd ./frontend && npm run dev'

  $QDBUS_CMD org.kde.konsole-$! /Sessions/2 setTitle 1 'BACKEND'
  $QDBUS_CMD org.kde.konsole-$! /Sessions/2 runCommand 'cd ./backend && air dev'

  $QDBUS_CMD org.kde.konsole-$! /Windows/2 org.kde.konsole.Window.setCurrentSession 1
  $QDBUS_CMD org.kde.konsole-$! /Sessions/0 setTitle 1 'GERP'

else # comando para WINDOWS

   echo 'nada configurado'
   echo 'probar cmder ==> https://www.eventslooped.com/posts/automate-open-tabs-in-cmder/'

fi
