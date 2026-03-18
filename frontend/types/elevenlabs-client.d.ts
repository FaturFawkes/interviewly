declare module "@elevenlabs/client" {
  export type Mode = "listening" | "speaking";
  export type Status = "disconnected" | "connecting" | "connected";

  export type MessageEvent = {
    role: "agent" | "user" | string;
    message: string;
    event_id?: number;
  };

  export type ConversationSessionOptions = {
    signedUrl: string;
    connectionType?: "websocket" | "webrtc" | string;
    userId?: string;
    overrides?: {
      agent?: {
        prompt?: {
          prompt?: string;
        };
        firstMessage?: string;
        language?: string;
      };
      tts?: {
        voiceId?: string;
        speed?: number;
        stability?: number;
        similarityBoost?: number;
      };
      conversation?: {
        textOnly?: boolean;
      };
      client?: {
        source?: string;
        version?: string;
      };
    };
    dynamicVariables?: Record<string, string | number | boolean>;
    onConnect?: (payload: { conversationId?: string }) => void;
    onStatusChange?: (payload: { status: Status }) => void;
    onModeChange?: (payload: { mode: Mode }) => void;
    onError?: (message?: string) => void;
    onDisconnect?: (payload?: { reason?: string; code?: number }) => void;
    onMessage?: (payload: MessageEvent) => void;
  };

  export class Conversation {
    static startSession(options: ConversationSessionOptions): Promise<Conversation>;
    sendContextualUpdate(message: string): void;
    endSession(): Promise<void>;
  }
}
