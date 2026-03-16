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
    connectionType?: "websocket" | string;
    userId?: string;
    onConnect?: (payload: { conversationId?: string }) => void;
    onStatusChange?: (payload: { status: Status }) => void;
    onModeChange?: (payload: { mode: Mode }) => void;
    onError?: (message?: string) => void;
    onDisconnect?: () => void;
    onMessage?: (payload: MessageEvent) => void;
  };

  export class Conversation {
    static startSession(options: ConversationSessionOptions): Promise<Conversation>;
    sendContextualUpdate(message: string): void;
    endSession(): Promise<void>;
  }
}
