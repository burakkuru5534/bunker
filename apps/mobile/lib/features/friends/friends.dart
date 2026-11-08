/// `friends` feature'ının diğer katmanlara açık giriş noktası.
library;

export 'package:siginak/features/friends/data/friends_repository.dart'
    show
        FriendPerson,
        FriendsRepository,
        Inbox,
        RoomInvite,
        friendsProvider,
        friendsRepositoryProvider,
        inboxProvider;
export 'package:siginak/features/friends/presentation/friends_screen.dart'
    show FriendsScreen;
export 'package:siginak/features/friends/presentation/invite_friends_sheet.dart'
    show showInviteFriendsSheet;
