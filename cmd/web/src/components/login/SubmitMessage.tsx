import React from "react";
import {
  Box,
  Alert,
  AlertIcon,
  AlertDescription,
  AlertProps,
} from "@chakra-ui/core";

export interface Props {
  message: string;
}

const SubmitMessage = ({ message, status }: Props & AlertProps) => {
  return (
    <Box my={4}>
      <Alert status={status} borderRadius={4}>
        <AlertIcon />
        <AlertDescription>{message}</AlertDescription>
      </Alert>
    </Box>
  );
};

export default SubmitMessage;
