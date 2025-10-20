// ABOUTME: Development mode utilities for testing different session phases
// ABOUTME: Provides fake data and state initialization for UI development

export function checkForDevMode(context) {
  const urlParams = new URLSearchParams(window.location.search);
  const devMode = urlParams.get('dev');

  if (devMode === 'true') {
    const phase = urlParams.get('phase') || 'lobby';
    const participantCount = parseInt(urlParams.get('participants') || '3');

    console.log(`[DEV MODE] Initializing ${phase} phase with ${participantCount} participants`);
    initDevSession(context, phase, participantCount);
  }
}

function initDevSession(context, phase, participantCount) {
  // Generate fake participants
  context.participants = generateFakeParticipants(participantCount);
  context.myId = context.participants[0].id;
  context.userName = context.participants[0].name;
  context.sessionCode = 'DEV123';
  context.isHost = true;
  context.connected = true; // Fake connection

  // Initialize phase-specific state
  switch (phase) {
    case 'lobby':
      context.currentView = 'lobby';
      break;

    case 'writing':
      context.currentView = 'writing';
      context.totalNotesNeeded = context.participants.length - 1;
      context.currentNoteIndex = 0;
      context.hasSubmittedNotes = false;
      context.updateNotesProgress();
      break;

    case 'reading':
      context.currentView = 'reading';
      context.currentReader = context.participants[0];
      context.isMyTurn = true;
      const totalNotesCount = participantCount * (participantCount - 1);
      context.totalNotes = totalNotesCount;
      context.notesRemaining = totalNotesCount;
      // Generate some fake notes for testing
      generateFakeNotes(context);
      break;

    case 'complete':
      context.currentView = 'reading';
      context.sessionComplete = true;
      generateFakeReceivedNotes(context);
      break;

    default:
      console.warn(`[DEV MODE] Unknown phase: ${phase}`);
      context.currentView = 'lobby';
  }
}

function generateFakeParticipants(count) {
  const names = ['Winter', 'Willow', 'Spring', 'Salix', 'Summer', 'Rose', 'Aster'];
  const participants = [];

  for (let i = 0; i < Math.min(count, names.length); i++) {
    participants.push({
      id: `dev-user-${i}`,
      name: names[i],
      isHost: i === 0
    });
  }

  return participants;
}

function generateFakeNotes(context) {
  // Generate fake notes for the reading phase
  const fakeNoteContents = [
    "I really appreciate your positive attitude and how you always bring energy to the team!",
    "Thank you for being such a great listener and always making time for others.",
    "Your attention to detail and thoroughness in your work is truly admirable.",
    "I'm grateful for your willingness to help others and share your knowledge.",
    "Your creative problem-solving skills have helped us overcome many challenges."
  ];

  context.currentNote = {
    id: 'dev-note-1',
    recipient: context.participants[0].name,
    recipientId: context.participants[0].id,
    content: fakeNoteContents[0]
  };
}

function generateFakeReceivedNotes(context) {
  const fakeNoteContents = [
    "I really appreciate your positive attitude and how you always bring energy to the team!",
    "Thank you for being such a great listener and always making time for others.",
    "Your attention to detail and thoroughness in your work is truly admirable."
  ];

  context.receivedNotes = fakeNoteContents.map((content, index) => ({
    id: `dev-received-${index}`,
    recipientId: context.myId,
    content: content
  }));
}
