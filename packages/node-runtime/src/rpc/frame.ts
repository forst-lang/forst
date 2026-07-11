import type { Writable } from "node:stream";
import * as Errors from "./errors.js";
import * as FrameErrors from "./frame_errors.js";

export const DEFAULT_MAX_MESSAGE_BYTES = 16 * 1024 * 1024;

export interface WireRequest {
  method: string;
  payloadJson: Uint8Array;
}

export interface ErrorDetail {
  code: number;
  message: string;
  dataJson?: Uint8Array;
}

export interface WireResponse {
  okJson?: Uint8Array;
  err?: ErrorDetail;
}

export interface Frame {
  id: number;
  request?: WireRequest;
  response?: WireResponse;
}

function encodeInt32Varint(value: number): Uint8Array {
  value = value | 0;
  const u =
    value >= 0
      ? BigInt(value)
      : BigInt.asUintN(64, BigInt(value));
  const bytes: number[] = [];
  let v = u;
  while (v >= 0x80n) {
    bytes.push(Number((v & 0x7fn) | 0x80n));
    v >>= 7n;
  }
  bytes.push(Number(v));
  return Uint8Array.from(bytes);
}

function varintToInt32(value: number): number {
  const u32 = value >>> 0;
  if (u32 >= 0x80000000) {
    return u32 - 0x100000000;
  }
  return u32 | 0;
}

function decodeSignedVarint(
  buf: Uint8Array,
  offset: number
): { value: number; next: number } {
  let result = 0n;
  let shift = 0n;
  let pos = offset;
  while (pos < buf.length) {
    const byte = BigInt(buf[pos]!);
    result |= (byte & 0x7fn) << shift;
    pos += 1;
    if ((byte & 0x80n) === 0n) {
      break;
    }
    shift += 7n;
    if (shift > 70n) {
      throw FrameErrors.protoVarintOverflow();
    }
  }
  const u32 = Number(result & 0xffffffffn);
  return { value: varintToInt32(u32), next: pos };
}

function encodeVarint(value: number): Uint8Array {
  const bytes: number[] = [];
  let v = value >>> 0;
  while (v >= 0x80) {
    bytes.push((v & 0x7f) | 0x80);
    v >>>= 7;
  }
  bytes.push(v);
  return Uint8Array.from(bytes);
}

function decodeVarint(
  buf: Uint8Array,
  offset: number
): { value: number; next: number } {
  let value = 0;
  let shift = 0;
  let pos = offset;
  while (pos < buf.length) {
    const byte = buf[pos]!;
    value |= (byte & 0x7f) << shift;
    pos += 1;
    if ((byte & 0x80) === 0) {
      return { value: value >>> 0, next: pos };
    }
    shift += 7;
    if (shift > 35) {
      throw FrameErrors.protoVarintOverflow();
    }
  }
  throw FrameErrors.protoTruncatedVarint();
}

function encodeField(tag: number, value: Uint8Array): Uint8Array {
  const out = new Uint8Array(1 + value.length);
  out[0] = tag;
  out.set(value, 1);
  return out;
}

function encodeInt32Field(fieldNumber: number, value: number): Uint8Array {
  const tag = (fieldNumber << 3) | 0;
  return encodeField(tag, encodeInt32Varint(value));
}

function encodeUint64Field(fieldNumber: number, value: number): Uint8Array {
  const tag = (fieldNumber << 3) | 0;
  return encodeField(tag, encodeVarint(value));
}

function encodeBytesField(fieldNumber: number, value: Uint8Array): Uint8Array {
  const tag = (fieldNumber << 3) | 2;
  const len = encodeVarint(value.length);
  const out = new Uint8Array(1 + len.length + value.length);
  out[0] = tag;
  out.set(len, 1);
  out.set(value, 1 + len.length);
  return out;
}

function encodeStringField(fieldNumber: number, value: string): Uint8Array {
  return encodeBytesField(fieldNumber, new TextEncoder().encode(value));
}

function encodeMessageField(fieldNumber: number, message: Uint8Array): Uint8Array {
  return encodeBytesField(fieldNumber, message);
}

function concatChunks(chunks: Uint8Array[]): Uint8Array {
  const total = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
  const out = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) {
    out.set(chunk, offset);
    offset += chunk.length;
  }
  return out;
}

function encodeWireRequest(req: WireRequest): Uint8Array {
  const chunks: Uint8Array[] = [encodeStringField(1, req.method)];
  if (req.payloadJson.length > 0) {
    chunks.push(encodeBytesField(2, req.payloadJson));
  }
  return concatChunks(chunks);
}

function encodeErrorDetail(err: ErrorDetail): Uint8Array {
  const chunks: Uint8Array[] = [
    encodeInt32Field(1, err.code),
    encodeStringField(2, err.message),
  ];
  if (err.dataJson !== undefined && err.dataJson.length > 0) {
    chunks.push(encodeBytesField(3, err.dataJson));
  }
  return concatChunks(chunks);
}

function encodeWireResponse(resp: WireResponse): Uint8Array {
  if (resp.err !== undefined) {
    return encodeMessageField(2, encodeErrorDetail(resp.err));
  }
  return encodeBytesField(1, resp.okJson ?? new Uint8Array(0));
}

export function encodeFrame(frame: Frame): Uint8Array {
  const chunks: Uint8Array[] = [];
  if (frame.id !== 0) {
    chunks.push(encodeUint64Field(1, frame.id));
  }
  if (frame.request !== undefined) {
    chunks.push(encodeMessageField(2, encodeWireRequest(frame.request)));
  } else if (frame.response !== undefined) {
    chunks.push(encodeMessageField(3, encodeWireResponse(frame.response)));
  }
  return concatChunks(chunks);
}

function readLengthDelimited(
  buf: Uint8Array,
  offset: number
): { value: Uint8Array; next: number } {
  const lenInfo = decodeVarint(buf, offset);
  const start = lenInfo.next;
  const end = start + lenInfo.value;
  if (end > buf.length) {
    throw FrameErrors.protoTruncatedLengthDelimitedField();
  }
  return { value: buf.subarray(start, end), next: end };
}

function decodeWireRequest(buf: Uint8Array): WireRequest {
  let method = "";
  let payloadJson = new Uint8Array(0);
  let pos = 0;
  while (pos < buf.length) {
    const tagInfo = decodeVarint(buf, pos);
    pos = tagInfo.next;
    const fieldNumber = tagInfo.value >>> 3;
    const wireType = tagInfo.value & 0x7;
    if (wireType !== 2) {
      throw FrameErrors.protoUnexpectedWireType("WireRequest", wireType);
    }
    const field = readLengthDelimited(buf, pos);
    pos = field.next;
    if (fieldNumber === 1) {
      method = new TextDecoder().decode(field.value);
    } else if (fieldNumber === 2) {
      payloadJson = new Uint8Array(field.value);
    } else {
      // Skip unknown fields.
    }
  }
  return { method, payloadJson };
}

function decodeErrorDetail(buf: Uint8Array): ErrorDetail {
  let code = 0;
  let message = "";
  let dataJson: Uint8Array | undefined;
  let pos = 0;
  while (pos < buf.length) {
    const tagInfo = decodeVarint(buf, pos);
    pos = tagInfo.next;
    const fieldNumber = tagInfo.value >>> 3;
    const wireType = tagInfo.value & 0x7;
    if (fieldNumber === 1 && wireType === 0) {
      const val = decodeSignedVarint(buf, pos);
      code = val.value;
      pos = val.next;
    } else if (wireType === 2) {
      const field = readLengthDelimited(buf, pos);
      pos = field.next;
      if (fieldNumber === 2) {
        message = new TextDecoder().decode(field.value);
      } else if (fieldNumber === 3) {
        dataJson = field.value;
      }
    } else if (wireType === 0) {
      const val = decodeVarint(buf, pos);
      pos = val.next;
    } else {
      throw FrameErrors.protoUnsupportedWireType("ErrorDetail", wireType);
    }
  }
  const detail: ErrorDetail = { code, message };
  if (dataJson !== undefined) {
    detail.dataJson = dataJson;
  }
  return detail;
}

function decodeWireResponse(buf: Uint8Array): WireResponse {
  let okJson: Uint8Array | undefined;
  let err: ErrorDetail | undefined;
  let pos = 0;
  while (pos < buf.length) {
    const tagInfo = decodeVarint(buf, pos);
    pos = tagInfo.next;
    const fieldNumber = tagInfo.value >>> 3;
    const wireType = tagInfo.value & 0x7;
    if (wireType !== 2) {
      throw FrameErrors.protoUnexpectedWireType("WireResponse", wireType);
    }
    const field = readLengthDelimited(buf, pos);
    pos = field.next;
    if (fieldNumber === 1) {
      okJson = field.value;
    } else if (fieldNumber === 2) {
      err = decodeErrorDetail(field.value);
    }
  }
  if (err !== undefined) {
    return { err };
  }
  return { okJson: okJson ?? new Uint8Array(0) };
}

export function decodeFrame(buf: Uint8Array): Frame {
  let id = 0;
  let request: WireRequest | undefined;
  let response: WireResponse | undefined;
  let pos = 0;
  while (pos < buf.length) {
    const tagInfo = decodeVarint(buf, pos);
    pos = tagInfo.next;
    const fieldNumber = tagInfo.value >>> 3;
    const wireType = tagInfo.value & 0x7;
    if (fieldNumber === 1 && wireType === 0) {
      const val = decodeVarint(buf, pos);
      id = val.value;
      pos = val.next;
    } else if (wireType === 2) {
      const field = readLengthDelimited(buf, pos);
      pos = field.next;
      if (fieldNumber === 2) {
        request = decodeWireRequest(field.value);
      } else if (fieldNumber === 3) {
        response = decodeWireResponse(field.value);
      }
    } else {
      throw FrameErrors.protoUnsupportedWireType("Frame", wireType);
    }
  }
  const frame: Frame = { id };
  if (request !== undefined) {
    frame.request = request;
  }
  if (response !== undefined) {
    frame.response = response;
  }
  return frame;
}

export function writeProtoFrame(
  stdout: Writable,
  frame: Frame,
  maxLen = DEFAULT_MAX_MESSAGE_BYTES
): void {
  const body = encodeFrame(frame);
  if (body.length > maxLen) {
    throw FrameErrors.protoFrameTooLarge(maxLen);
  }
  const header = Buffer.alloc(4);
  header.writeUInt32BE(body.length, 0);
  stdout.write(header);
  stdout.write(body);
}

export function newRequestFrame(
  id: number,
  method: string,
  params: unknown
): Frame {
  const payloadJson = new TextEncoder().encode(JSON.stringify(params ?? {}));
  return {
    id,
    request: { method, payloadJson },
  };
}

export function newOkResponseFrame(id: number, result: unknown): Frame {
  const okJson = new TextEncoder().encode(JSON.stringify(result ?? null));
  return {
    id,
    response: { okJson },
  };
}

export function newErrorResponseFrame(
  id: number,
  err: Errors.JsonRpcError
): Frame {
  const detail: ErrorDetail = {
    code: err.code,
    message: err.message,
  };
  if (err.data !== undefined) {
    detail.dataJson = new TextEncoder().encode(JSON.stringify(err.data));
  }
  return {
    id,
    response: { err: detail },
  };
}

export function parseRequestFrame(frame: Frame): {
  id: number;
  method: string;
  params: unknown;
} {
  if (frame.request === undefined) {
    throw Errors.frameMissingRequest();
  }
  const { method, payloadJson } = frame.request;
  if (method === "") {
    throw Errors.requestMethodRequired();
  }
  if (payloadJson.length === 0) {
    return { id: frame.id, method, params: undefined };
  }
  try {
    return { id: frame.id, method, params: JSON.parse(new TextDecoder().decode(payloadJson)) };
  } catch {
    throw Errors.invalidRequestPayloadJson();
  }
}

export class ProtoFrameReader {
  private buffer = Buffer.alloc(0);

  append(chunk: Buffer | Uint8Array): void {
    this.buffer = Buffer.concat([this.buffer, Buffer.from(chunk)]);
  }

  tryReadFrame(maxLen = DEFAULT_MAX_MESSAGE_BYTES): Frame | null {
    if (this.buffer.length < 4) {
      return null;
    }
    const bodyLen = this.buffer.readUInt32BE(0);
    if (bodyLen === 0) {
      throw FrameErrors.protoEmptyFrame();
    }
    if (bodyLen > maxLen) {
      throw FrameErrors.protoFrameTooLarge(maxLen);
    }
    const totalLen = 4 + bodyLen;
    if (this.buffer.length < totalLen) {
      return null;
    }
    const body = this.buffer.subarray(4, totalLen);
    this.buffer = this.buffer.subarray(totalLen);
    return decodeFrame(body);
  }
}

export async function readProtoFrameFromStream(
  stream: NodeJS.ReadableStream,
  maxLen = DEFAULT_MAX_MESSAGE_BYTES
): Promise<Frame | null> {
  const reader = new ProtoFrameReader();
  for await (const chunk of stream) {
    reader.append(Buffer.from(chunk as Buffer));
    const frame = reader.tryReadFrame(maxLen);
    if (frame !== null) {
      return frame;
    }
  }
  return reader.tryReadFrame(maxLen);
}
