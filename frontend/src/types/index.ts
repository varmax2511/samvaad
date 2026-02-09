export interface User {
  id: string;
  username: string;
  token: string;
}

export interface Message {
  type: string;
  roomId?: string;
  from?: string;
  to?: string;
  payload?: unknown;
}

export interface JoinPayload {
  userId: string;
  otherUsers: string[];
}

export interface UserPayload {
  userId: string;
}

export interface ErrorPayload {
  message: string;
}

export interface RTCPayload {
  sdp?: RTCSessionDescriptionInit;
  candidate?: RTCIceCandidateInit;
}
