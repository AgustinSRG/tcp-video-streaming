

export function renderTimeSeconds(s: number): string {
    if (isNaN(s) || !isFinite(s) || s < 0) {
        s = 0;
    }
    s = Math.floor(s);
    let hours = 0;
    let minutes = 0;
    if (s >= 3600) {
        hours = Math.floor(s / 3600);
        s = s % 3600;
    }
    if (s > 60) {
        minutes = Math.floor(s / 60);
        s = s % 60;
    }
    let r = "";

    if (s > 9) {
        r = "" + s + r;
    } else {
        r = "0" + s + r;
    }

    if (minutes > 9) {
        r = "" + minutes + ":" + r;
    } else {
        r = "0" + minutes + ":" + r;
    }

    if (hours > 0) {
        if (hours > 9) {
            r = "" + hours + ":" + r;
        } else {
            r = "0" + hours + ":" + r;
        }
    }

    return r;
}

export function parseTimeSeconds(str: string): number {
    const parts = (str || "").trim().split(":");

    let h = 0;
    let m = 0;
    let s = 0;

    if (parts.length > 2) {
        h = parseInt(parts[0], 10) || 0;
        m = parseInt(parts[1], 10) || 0;
        s = parseInt(parts[2], 10) || 0;
    } else if (parts.length > 1) {
        m = parseInt(parts[0], 10) || 0;
        s = parseInt(parts[1], 10) || 0;
    } else {
        s = parseInt(parts[0], 10) || 0;
    }

    return (h * 3600) + (m * 60) + s;
}

