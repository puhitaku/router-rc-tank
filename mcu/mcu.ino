#include <Ticker.h>

// モーターコントローラーに接続されているピンの番号
int a1 = 5, a2 = 4, apwm = 3;
int b1 = 8, b2 = 7, bpwm = 6;
int stby = 2;

// PWM デューティ比設定
const int max_duty = 128;

int a_duty = 0;
int b_duty = 0;

// 操作を表す enum
typedef enum Operation {
    stop,
    forward,
    backward,
    turn_right,
    turn_left,
} operation_t;

// 起動時は停止
operation_t operation = stop;

// PWM 出力調整関数のプロトタイプ宣言
void adjust_output();

// 10ms ごとに PWM 出力を調整
Ticker tick(adjust_output, 10);

// 起動時に一度呼ばれる関数
void setup() {
    // PWM の周波数が元々低いので 31.4kHz に上げる
    // http://rtmrw.parallel.jp/laboratory6/lab-report-152/lab-152.html
    TCA0.SINGLE.CTRLA = 0b111;

    pinMode(a1, OUTPUT);
    pinMode(a2, OUTPUT);
    pinMode(apwm, OUTPUT);

    pinMode(b1, OUTPUT);
    pinMode(b2, OUTPUT);
    pinMode(bpwm, OUTPUT);

    pinMode(stby, OUTPUT);
    digitalWrite(stby, HIGH);

    Serial.begin(115200);
    delay(100);
    tick.start();
}

// 無限ループする関数
void loop() {
    tick.update();

    // もしシリアルで何かを受信したら、その文字に応じて現在の操作を変更する
    if (Serial.available() > 0) {
        switch (Serial.read()) {
        case 's':
            operation = stop;
            break;
        case 'f':
            operation = forward;
            break;
        case 'b':
            operation = backward;
            break;
        case 'r':
            operation = turn_right;
            break;
        case 'l':
            operation = turn_left;
            break;
        }
    }
}

// from から to へ向かって diff だけ変化させた値を返す
// もし from と to が等しかった場合はそのまま返す
int interpolate(int from, int to, int diff) {
    if (from == to) {
        return from;
    } else if (from < to) {
        if (from + diff > to) {
            from = to;
        } else {
            from += diff;
        }
    } else {
        if (from - diff < to) {
            from = to;
        } else {
            from -= diff;
        }
    }
    return from;
}

// 現在の操作に応じて PWM 出力を調整する
// 過大な電流を防ぐため、モーターは少しずつ始動・停止する
void adjust_output() {
    int to_a, to_b;

    switch (operation) {
    case stop:
        to_a = 0;
        to_b = 0;
        break;
    case forward:
        to_a = max_duty;
        to_b = max_duty;
        break;
    case backward:
        to_a = -max_duty;
        to_b = -max_duty;
        break;
    case turn_right:
        to_a = -max_duty;
        to_b = max_duty;
        break;
    case turn_left:
        to_a = max_duty;
        to_b = -max_duty;
        break;
    }

    // デューティ比を5ずつ変化させる
    a_duty = interpolate(a_duty, to_a, 5);
    b_duty = interpolate(b_duty, to_b, 5);

    // 現在のデューティ比の符号に合わせてモーターの回転方向を設定する
    if (a_duty >= 0) {
        digitalWrite(a1, HIGH);
        digitalWrite(a2, LOW);
    } else {
        digitalWrite(a1, LOW);
        digitalWrite(a2, HIGH);
    }

    if (b_duty >= 0) {
        digitalWrite(b1, HIGH);
        digitalWrite(b2, LOW);
    } else {
        digitalWrite(b1, LOW);
        digitalWrite(b2, HIGH);
    }

    // デューティ比を設定
    analogWrite(apwm, abs(a_duty));
    analogWrite(bpwm, abs(b_duty));
}
