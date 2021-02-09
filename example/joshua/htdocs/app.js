'use strict';

var player = 1;
var lineColor = "#7D96BF";

var canvas = document.getElementById('tic-tac-toe-board');
var context = canvas.getContext('2d');

var canvasSize = 500;
var sectionSize = canvasSize / 3;
canvas.width = canvasSize;
canvas.height = canvasSize;
context.translate(0.5, 0.5);

function getInitialBoard(defaultValue) {
    var board = [];

    for (var x = 0; x < 3; x++) {
        board.push([]);

        for (var y = 0; y < 3; y++) {
            board[x].push(defaultValue);
        }
    }

    return board;
}

var board = getInitialBoard("");

function addPlayingPiece(mouse) {
    var xCordinate;
    var yCordinate;

    for (var x = 0; x < 3; x++) {
        for (var y = 0; y < 3; y++) {
            xCordinate = x * sectionSize;
            yCordinate = y * sectionSize;

            if (
                mouse.x >= xCordinate && mouse.x <= xCordinate + sectionSize &&
                mouse.y >= yCordinate && mouse.y <= yCordinate + sectionSize
            ) {

                clearPlayingArea(xCordinate, yCordinate);

                if (player === 1) {
                    drawX(xCordinate, yCordinate);
                } else {
                    drawO(xCordinate, yCordinate);
                }
            }
        }
    }
}

function clearPlayingArea(xCordinate, yCordinate) {
    context.fillStyle = "#191F2B";
    context.fillRect(
        xCordinate+10,
        yCordinate+10,
        sectionSize-20,
        sectionSize-20
    );
}
function drawO(xCordinate, yCordinate) {
    var halfSectionSize = (0.5 * sectionSize);
    var centerX = xCordinate + halfSectionSize;
    var centerY = yCordinate + halfSectionSize;
    var radius = (sectionSize - 80) / 2;
    var startAngle = 0 * Math.PI;
    var endAngle = 2 * Math.PI;

    context.lineWidth = 14;
    context.strokeStyle = "#7D96BF";
    context.beginPath();
    context.arc(centerX, centerY, radius, startAngle, endAngle);
    context.stroke();
}

function drawX(xCordinate, yCordinate) {
    context.strokeStyle = "#7D96BF";

    context.beginPath();

    var offset = 40;
    context.moveTo(xCordinate + offset, yCordinate + offset);
    context.lineTo(xCordinate + sectionSize - offset, yCordinate + sectionSize - offset);

    context.moveTo(xCordinate + offset, yCordinate + sectionSize - offset);
    context.lineTo(xCordinate + sectionSize - offset, yCordinate + offset);
    context.lineWidth = 15;


    context.stroke();
}

function drawLines(lineWidth, strokeStyle) {
    var lineStart = 4;
    var lineLenght = canvasSize - 5;
    context.lineWidth = lineWidth;
    context.lineCap = 'round';
    context.strokeStyle = strokeStyle;
    context.beginPath();

    /*
     * Horizontal lines 
     */
    for (var y = 1; y <= 2; y++) {
        context.moveTo(lineStart, y * sectionSize);
        context.lineTo(lineLenght, y * sectionSize);
    }

    /*
     * Vertical lines 
     */
    for (var x = 1; x <= 2; x++) {
        context.moveTo(x * sectionSize, lineStart);
        context.lineTo(x * sectionSize, lineLenght);
    }

    context.stroke();
}

drawLines(10, lineColor);

function getCanvasMousePosition(event) {
    var rect = canvas.getBoundingClientRect();

    return {
        x: event.clientX - rect.left,
        y: event.clientY - rect.top
    }
}

canvas.addEventListener('mouseup', function (event) {
    if (player === 1) {
        player = 2;
    } else {
        player = 1;
    }

    var canvasMousePosition = getCanvasMousePosition(event);
    addPlayingPiece(canvasMousePosition);
    drawLines(10, lineColor);
});


var ws = new WebSocket("ws://localhost:8080/ws");

ws.onmessage = function (event) {
    var messages = event.data.split('\n');
    for (var i = 0; i < messages.length; i++) {
        var message = JSON.parse(messages[i]);
        onMessage(message);
    }
};

function onMessage(message) {
    var xCordinate;
    var yCordinate;
    switch (message.Player) {
        case 1:
            if (0 <= message.Single && message.Single < 3) {
                xCordinate = 0;
                yCordinate = message.Single;
            }
            if (3 <= message.Single && message.Single < 6) {
                xCordinate = 1;
                yCordinate = message.Single - 3;
            }
            if (6 <= message.Single && message.Single < 9) {
                xCordinate = 2;
                yCordinate = message.Single - 6;
            }
            //clearPlayingArea(xCordinate, yCordinate);
            drawX(xCordinate * sectionSize, yCordinate * sectionSize);
            break;
        case 2:
            if (0 <= message.Single && message.Single < 3) {
                xCordinate = 0;
                yCordinate = message.Single;
            }
            if (3 <= message.Single && message.Single < 6) {
                xCordinate = 1;
                yCordinate = message.Single - 3;
            }
            if (6 <= message.Single && message.Single < 9) {
                xCordinate = 2;
                yCordinate = message.Single - 6;
            }
            drawO(xCordinate * sectionSize, yCordinate * sectionSize);
            break;
        default:
            for (var x = 0; x < 3; x++) {
                for (var y = 0; y < 3; y++) {
                    clearPlayingArea(x * sectionSize, y * sectionSize);
                }
            }
            break;
    }

}