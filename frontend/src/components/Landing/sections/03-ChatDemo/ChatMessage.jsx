import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { Bot, User, AlertTriangle, CheckCircle } from 'lucide-react';

/**
 * Chat message component with typing effect
 */
const ChatMessage = ({ message, isUser, isTyping, messageType, onProgress }) => {
  const [displayedText, setDisplayedText] = useState('');
  const [isTypingAnimation, setIsTypingAnimation] = useState(!isUser);

  useEffect(() => {
    if (isUser || isTyping) {
      setDisplayedText(message);
      setIsTypingAnimation(false);
      onProgress?.();
      return;
    }

    // Typing animation for AI messages
    let currentIndex = 0;
    setDisplayedText('');
    setIsTypingAnimation(true);

    const typingInterval = setInterval(() => {
      if (currentIndex < message.length) {
        setDisplayedText(message.substring(0, currentIndex + 1));
        currentIndex++;
        onProgress?.();
      } else {
        setIsTypingAnimation(false);
        onProgress?.();
        clearInterval(typingInterval);
      }
    }, 15); // 15ms per character for smooth typing

    return () => clearInterval(typingInterval);
  }, [message, isUser, isTyping, onProgress]);

  const getTypeIcon = () => {
    switch (messageType) {
      case 'error': return <AlertTriangle size={14} className="msg-icon error" />;
      case 'warning': return <AlertTriangle size={14} className="msg-icon warning" />;
      case 'consequence': return <AlertTriangle size={14} className="msg-icon error" />;
      case 'solution': return <CheckCircle size={14} className="msg-icon success" />;
      case 'benefit': return <CheckCircle size={14} className="msg-icon success" />;
      case 'positive': return <CheckCircle size={14} className="msg-icon success" />;
      default: return null;
    }
  };

  return (
    <motion.div 
      className={`chat-message ${isUser ? 'chat-message--user' : 'chat-message--ai'}`}
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
    >
      <div className="chat-message__avatar">
        {isUser ? <User size={16} /> : <Bot size={16} />}
      </div>
      <div className={`chat-message__content ${messageType ? `chat-message__content--${messageType}` : ''}`}>
        {isTyping ? (
          <div className="typing-indicator">
            <span></span><span></span><span></span>
          </div>
        ) : (
          <div className="message-flex-container">
            {getTypeIcon()}
            <span className="message-text">
              {displayedText}
              {isTypingAnimation && <span className="typing-cursor">|</span>}
            </span>
          </div>
        )}
      </div>
    </motion.div>
  );
};

export default ChatMessage;
