import { wsManager } from './socket.js';
import { patcher } from '../engine/patcher.js';

export class WSDispatcher {
  static init() {
    wsManager.onMessage((msg) => {
      WSDispatcher.handleMessage(msg);
    });
  }

  static handleMessage(msg) {
    if (!msg || !msg.event) return;

    // 1. Structured Batch Event (e.g. Sales Order transaction modifying multiple items and stocks)
    if (msg.event === 'BATCH_TRANSACTION') {
      const batch = msg.payload;
      patcher.handleBatchTransaction(batch);
      return;
    }

    // 2. Atomic Events
    switch (msg.event) {
      case 'ENTITY_CREATED':
        if (msg.topic === 'products') {
          patcher.handleProductCreated(msg.payload);
        } else if (msg.topic === 'orders') {
          patcher.handleOrderCreated(msg.payload);
        } else if (msg.topic === 'audit') {
          patcher.handleAuditLogCreated(msg.payload);
        }
        break;

      case 'ENTITY_UPDATED':
        if (msg.topic === 'products') {
          patcher.handleProductUpdated(msg.payload);
        }
        break;

      case 'ENTITY_DELETED':
        if (msg.topic === 'products') {
          patcher.handleProductDeleted(msg.payload);
        }
        break;

      case 'STOCK_CHANGED':
        if (msg.topic === 'products') {
          patcher.handleStockChanged(msg.payload);
        }
        break;

      case 'SYSTEM_BROADCAST':
        console.log('[System Broadcast]', msg.payload);
        break;

      default:
        console.debug('Unhandled event:', msg.event, msg);
    }
  }
}
