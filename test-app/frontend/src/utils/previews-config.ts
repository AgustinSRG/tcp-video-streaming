/* Preview configuration utilities */

"use strict";

export interface PreviewsConfiguration {
    enabled: boolean;
    width: number;
    height: number;
    delaySeconds: number;
}

export function parsePreviewsConfiguration(str: string): PreviewsConfiguration {
    str = str.trim().toLowerCase();

    if (!str || str === "false") {
        return {
            enabled: false,
            width: 0,
            height: 0,
            delaySeconds: 0,
        };
    }

    let parts = str.split(",");

    let delay = parts[1] ? (parseInt((parts[1] + "").trim(), 10)) : -1;

    parts = (parts[0] + "").trim().split("x");

    let width = parts[0] ? (parseInt((parts[0] + "").trim(), 10)) : -1;
    let height = parts[1] ? (parseInt((parts[1] + "").trim(), 10)) : -1;

    if (delay <= 0 || width <= 0 || height <= 0) {
        return {
            enabled: false,
            width: 0,
            height: 0,
            delaySeconds: 0,
        };
    }

    return {
        enabled: true,
        width: width,
        height: height,
        delaySeconds: delay,
    };
}

export function encodePreviewsConfiguration(pc: PreviewsConfiguration): string {
    if (pc.enabled) {
        return pc.width + "x" + pc.height + "," + pc.delaySeconds;
    } else {
        return "False";
    }
}