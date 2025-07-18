import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import { chatKeys } from '../lib/queryKeys';
import { CapturedStateUnit, ChatMessage, ChatSession } from '../lib/types';

export function useChats() {
  return useQuery<ChatSession[]>({
    queryKey: chatKeys.all,
    queryFn: api.getChats,
  });
}

export function useCreateChat() {
  const queryClient = useQueryClient();
  return useMutation<Partial<ChatSession>, Error, Partial<ChatSession>>({
    mutationFn: api.createChat,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: chatKeys.all });
    },
  });
}

export function useChatHistory(id: string, options?: { enabled?: boolean }) {
  return useQuery<ChatMessage[]>({
    queryKey: chatKeys.history(id),
    queryFn: () => api.getChatHistory(id),
    enabled: options?.enabled ?? !!id,
  });
}

export function useSendMessage(chatId: string) {
  const queryClient = useQueryClient();
  return useMutation<
    { response: string; state: CapturedStateUnit[] },
    Error,
    { message: string; provider?: string; models?: string[] }
  >({
    mutationFn: ({ message, provider, models }) =>
      api.sendMessage(chatId, message, provider, models),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: chatKeys.history(chatId) });
    },
  });
}

export function useChatInstruction(chatId: string) {
  const queryClient = useQueryClient();
  return useMutation<ChatMessage[], Error, string>({
    mutationFn: instruction => api.sendInstruction(chatId, instruction),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: chatKeys.history(chatId) });
    },
  });
}
