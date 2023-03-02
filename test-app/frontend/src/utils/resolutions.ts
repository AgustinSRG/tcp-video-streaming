/* Resolutions utilities */

"use strict";

export interface Resolution {
    width: number;
    height: number;
    fps: number;
}

export interface ResolutionList {
    hasOriginal: boolean;
    resolutions: Resolution[];
}

export function parseResolution(str: string): Resolution {
    let parts = str.split("-");
    let fps = parts[1] ? (parseInt((parts[1] + "").trim(), 10)) : -1;
    if (fps <= 0) {
        fps = -1;
    }
    parts = (parts[0] + "").toLowerCase().split("x");
    const width = parts[0] ? (parseInt((parts[0] + "").trim(), 10)) : 0;
    const height = parts[1] ? (parseInt((parts[1] + "").trim(), 10)) : 0;

    return {
        width: width,
        height: height,
        fps: fps,
    };
}

export function encodeResolution(r: Resolution): string {
    if (r.fps <= 0) {
        return r.width + "x" + r.height;
    } else {
        return r.width + "x" + r.height + "-" + r.fps;
    }
}

export function parseResolutionList(str: string): ResolutionList{
    str = str.trim();

    if (!str) {
        return {
            hasOriginal: true,
            resolutions: [],
        };
    }

    const res: ResolutionList = {
        hasOriginal: false,
        resolutions: [],
    }

    const parts = str.split(",");

    for (let part of parts) {
        part = part.trim();

        if (!part) {
            continue;
        }

        if (part.toUpperCase() === "ORIGINAL") {
            res.hasOriginal = true;
            continue;
        }
        const r = parseResolution(part);
        if (r.width <= 0 || r.height <= 0) {
            continue;
        }

        res.resolutions.push(r);
    }

    return res;
}

export function encodeResolutionList(rl: ResolutionList): string {
    if (rl.hasOriginal) {
        return "ORIGINAL" + ((rl.resolutions.length > 0) ? ("," + rl.resolutions.map(encodeResolution).join(",")): "");
    } else {
        return rl.resolutions.map(encodeResolution).join(",");
    }
}

export function getVODIndex(url: string): number {
    const parts = (url + "").split("/");
    const fileName = (parts.pop() + "").split(".")[0] + "";
    const fileIndex = parseInt(fileName.split("-")[1] + "", 10) || 0;
    return fileIndex;
}
