declare module "mammoth" {
  export type ExtractRawTextInput = {
    arrayBuffer: ArrayBuffer;
  };

  export type ExtractRawTextResult = {
    value: string;
  };

  export function extractRawText(input: ExtractRawTextInput): Promise<ExtractRawTextResult>;

  const mammoth: {
    extractRawText: typeof extractRawText;
  };

  export default mammoth;
}
