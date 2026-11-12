import { traderApi } from './traders'
import { strategyApi } from './strategies'
import { configApi } from './config'
import { dataApi } from './data'
import { telegramApi } from './telegram'
import { walletApi } from './wallet'
import * as chatApi from './chat'
import * as chatSessionsApi from './chatSessions'
import * as notifyChannelsApi from './notifyChannels'

export const api = {
  ...traderApi,
  ...strategyApi,
  ...configApi,
  ...dataApi,
  ...telegramApi,
  ...walletApi,
  listChatTools: chatApi.listChatTools,
  testChatTool: chatApi.testChatTool,
  chatCompletions: chatApi.chatCompletions,
  streamChat: chatApi.streamChat,
  listChatSessions: chatSessionsApi.listChatSessions,
  createChatSession: chatSessionsApi.createChatSession,
  getChatSession: chatSessionsApi.getChatSession,
  renameChatSession: chatSessionsApi.renameChatSession,
  deleteChatSession: chatSessionsApi.deleteChatSession,
  getNotifyChannel: notifyChannelsApi.getNotifyChannel,
  saveNotifyChannel: notifyChannelsApi.saveNotifyChannel,
  unbindNotifyChannel: notifyChannelsApi.unbindNotifyChannel,
  clearNotifyChannel: notifyChannelsApi.clearNotifyChannel,
}
